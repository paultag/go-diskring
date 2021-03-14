// {{{ Copyright (c) Paul R. Tagliamonte <paultag@gmail.com> 2020-2021
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE. }}}

package diskring

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"unsafe"
)

// Cursor is the opaque set of two uintptrs used to store the head and tail
// information.
type Cursor struct {
	head uintptr
	tail uintptr
}

// Ring contains internal state backing the actual diskring. This works by
// mmapping a file into the Ring, and aligning it so that reads and writes
// below the size of the buffer wrap.
type Ring struct {
	file *os.File

	ringBase uintptr
	ringOne  uintptr
	ringTwo  uintptr

	size uintptr

	headerBase uintptr
	headerSize uintptr
	cursor     *Cursor

	buf []byte

	blockWrites bool
	mutex       sync.Mutex
}

// New will create a new Ring Buffer using the underlying file
// (`fd`) to read and write entries to. Ensure that the file was opened r/w and
// the user is able to mmap the file.
func New(fd *os.File) (*Ring, error) {
	return NewWithOptions(fd, Options{
		// Offset is 0
	})
}

// Open will open the existing file at the provided path, and return it
// as a loaded Ring buffer.
//
// In addition to the Ring buffer and any error conditions, this function also
// passes a closer to close the handle to the underlying *os.File object.
// This should be defer'd in a place that makes sense in the Ring lifecycle.
func Open(path string) (*Ring, error) {
	fd, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	ring, err := New(fd)
	if err != nil {
		fd.Close()
		return nil, err
	}
	return ring, nil
}

// OpenWithOptions will open the existing file at the provided path, and return it
// as a loaded Ring buffer.
//
// In addition to the Ring buffer and any error conditions, this function also
// passes a closer to close the handle to the underlying *os.File object.
// This should be defer'd in a place that makes sense in the Ring lifecycle.
//
// Additionally, this will construct the Ring according to the options
// set in the passed Options struct.
func OpenWithOptions(path string, options Options) (*Ring, error) {
	fd, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	ring, err := NewWithOptions(fd, options)
	if err != nil {
		fd.Close()
		return nil, err
	}
	return ring, nil
}

// Options contains some "extra" configuration that can be used to control
// the internals of the Ring. If you do not require these options, it's best
// to invoke New, and let the library take care of defaults.
type Options struct {

	// ReserveHeader will use the first page for a diskring "header" where
	// the cursor will be persisted to
	ReserveHeader bool

	// CustomHeader will create a custom header given the provided base address
	// and size (in bytes) within the diskring Header.
	//
	// The custom user-header may contain a diskring.Cursor in it, and if so,
	// returning a pointer to that object will provide state to diskring
	// internals.
	//
	// A nil value will mean using an in-memory cursor.
	CustomHeader func(unsafe.Pointer, int) (*Cursor, error)
}

// NewWithOptions will create a new Ring Buffer using the underlying file
// (`fd`) to read and write entries to. Ensure that the file was opened r/w and
// the user is able to mmap the file.
//
// Additionally, this will construct the Ring according to the options
// set in the passed Options struct.
func NewWithOptions(fd *os.File, options Options) (*Ring, error) {
	stat, err := fd.Stat()
	if err != nil {
		return nil, err
	}

	var (
		size             = uintptr(stat.Size())
		offset     int64 = 0
		cur              = &Cursor{head: 0, tail: 0}
		headerBase uintptr
	)
	if options.ReserveHeader {
		offset = int64(syscall.Getpagesize())
		size -= uintptr(offset)

		if offset <= int64(unsafe.Sizeof(Cursor{})) {
			return nil, fmt.Errorf("offset can't store cursor")
		}

		// the 1st argument ("offset") is actually the size, since
		// we're allocating the pre-offset fd hunk.
		headerBase, err = mmap(0, uintptr(offset),
			syscall.PROT_READ|syscall.PROT_WRITE,
			syscall.MAP_SHARED,
			int(fd.Fd()), 0)
		if err != nil {
			return nil, err
		}

		unsafeHeaderBase := unsafe.Pointer(headerBase)

		// OK, we have the header allocated and ready for use. Now let's
		// check if this is user controlled, or we can use it for our
		// cursor.

		if options.CustomHeader == nil {
			// If we don't have a custom header layout, we can go ahead
			// and use the whooooooooooooole 4k block for 2 uintptrs.
			cur = (*Cursor)(unsafeHeaderBase)
		} else {
			// Let's ask the user nicely to allocate us space for a
			// diskring.Cursor. If we get one, we can overwrite our
			// in-memory cursor.
			userCursor, err := options.CustomHeader(unsafeHeaderBase, int(offset))
			if err != nil {
				return nil, err
			}
			if userCursor != nil {
				cur = userCursor
			}
		}
	}

	if int(size)%syscall.Getpagesize() != 0 {
		return nil, fmt.Errorf("File must be aligned to page size")
	}

	// First, we need to mmap a chunk that's twice the size of the file that
	// we'll mmap, so that we can mmap two fixed offset blocks inside that
	// block.
	ringBase, err := mmap(0, size<<1,
		syscall.PROT_NONE,
		syscall.MAP_ANONYMOUS|syscall.MAP_PRIVATE,
		-1, offset)
	if err != nil {
		return nil, err
	}

	ringOne, err := mmap(ringBase, size,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_FIXED|syscall.MAP_SHARED, int(fd.Fd()), offset)
	if err != nil {
		return nil, err
	}

	if ringBase != ringOne {
		return nil, fmt.Errorf("mmap split our MAP_FIXED call")
	}

	ringTwo, err := mmap(ringBase+size, size,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_FIXED|syscall.MAP_SHARED, int(fd.Fd()), offset)
	if err != nil {
		return nil, err
	}
	if ringTwo != ringOne+size {
		return nil, fmt.Errorf("mmap split our mirror MAP_FIXED call")
	}

	return &Ring{
		file: fd,
		size: size,

		headerBase: headerBase,
		headerSize: uintptr(offset),
		cursor:     cur,

		ringBase: ringBase,
		ringOne:  ringOne,
		ringTwo:  ringTwo,

		buf: *asByteSlice(ringBase, int(size<<1)),

		mutex:       sync.Mutex{},
		blockWrites: false,
	}, nil
}

// Close will unmap all mapped memory, as well as close the underlying
// file handle.
func (r *Ring) Close() error {
	if r.headerBase != 0 {
		if err := munmap(r.headerBase, r.headerSize); err != nil {
			return err
		}
	}
	if err := munmap(r.ringOne, r.size); err != nil {
		return err
	}
	if err := munmap(r.ringTwo, r.size); err != nil {
		return err
	}
	if err := munmap(r.ringBase, r.size<<1); err != nil {
		return err
	}
	return r.file.Close()
}

// vim: foldmethod=marker
