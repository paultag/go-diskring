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
	"unsafe"
)

// BlockWrites will prevent any new writes from hitting the Ring. This will
// hang all writes, but it will allow a process to read the buffer completely,
// then unlock the buffer if taking new writes is absolutely unacceptable.
func (r *Ring) BlockWrites() {
	r.mutex.Lock()
	r.blockWrites = true
	r.mutex.Unlock()
}

// UnblockWrites will allow writes to the buffer after calling `BlockWrites`.
func (r *Ring) UnblockWrites() {
	r.mutex.Lock()
	r.blockWrites = false
	r.mutex.Unlock()
}

// Write a block of data into the disk ring. If there's not enough data in the
// diskring, this will advance the head until we can fit the data in. If the
// data is more than 1/4 the size of the ring, the write will fail because
// it's an arbitrary number I picked.
func (r *Ring) Write(buf []byte) (int, error) {
	if r.readOnly {
		return 0, fmt.Errorf("diskring: read only")
	}
	if len(buf) > int(r.size/4) {
		return 0, fmt.Errorf("diskring: data is too large")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	blen := uintptr(len(buf))
	for {
		if (blen + uintptrSize) > r.freeBytes() {
			if err := r.advanceHead(); err != nil {
				return 0, err
			}
			continue
		}
		break
	}

	m := copy(r.buf[r.cursor.tail+uintptrSize:], buf)
	*(*uintptr)(unsafe.Pointer(&r.buf[r.cursor.tail])) = uintptr(m)
	r.cursor.tail = ((r.cursor.tail + uintptrSize + uintptr(m)) % r.size)

	select {
	case r.wakeup <- struct{}{}:
	default:
	}

	return m, nil
}

// vim: foldmethod=marker
