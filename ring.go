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
)

// Ring contains internal state backing the actual diskring. This works by
// mmapping a file into the Ring, and aligning it so that reads and writes
// below the size of the buffer wrap.
type Ring struct {
	ringBase uintptr
	ringOne  uintptr
	ringTwo  uintptr

	buf []byte

	size uintptr
	head uintptr
	tail uintptr

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
func Open(path string) (*Ring, func() error, error) {
	fd, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, nil, err
	}
	ring, err := New(fd)
	if err != nil {
		fd.Close()
		return nil, nil, err
	}
	return ring, fd.Close, nil
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
func OpenWithOptions(path string, options Options) (*Ring, func() error, error) {
	fd, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, nil, err
	}
	ring, err := NewWithOptions(fd, options)
	if err != nil {
		fd.Close()
		return nil, nil, err
	}
	return ring, fd.Close, nil
}

// Options contains some "extra" configuration that can be used to control
// the internals of the Ring. If you do not require these options, it's best
// to invoke New, and let the library take care of defaults.
type Options struct {
	// Offset will set the number of bytes into the target file
	// before mapping the ring buffer. This can be useful if you require
	// a fixed header in the first N bytes of the file.
	Offset int64
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
	size := uintptr(stat.Size())

	// We need to account for the header in our allocations here.
	offset := options.Offset
	size -= uintptr(offset)

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
		size: size,
		head: 0,
		tail: 0,

		ringBase: ringBase,
		ringOne:  ringOne,
		ringTwo:  ringTwo,

		buf: *asByteSlice(ringBase, int(size<<1)),

		mutex:       sync.Mutex{},
		blockWrites: false,
	}, nil
}

// vim: foldmethod=marker
