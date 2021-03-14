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
	"io"
	"unsafe"
)

var (
	// We use the uintptr size (since it's a size the CPU can deal with
	// super well, and it's how we address memory anyway), and use that to
	// store offsets and sizes.
	uintptrSize = unsafe.Sizeof(uintptr(0))
)

// UNSAFE
//
// Read the head pointer's entry length, and jump ahead by that amount.
// This will set the head to the next entry, dropping the old head entry.
//
// This can either be used to reclaim space, or to advance the head pointer
// after processing the data.
//
func (r *Ring) advanceHead() error {
	if r.len() == 0 {
		return io.EOF
	}
	length := *(*uintptr)(unsafe.Pointer(&r.buf[r.head]))
	r.head = (r.head + length + uintptrSize) % r.size
	return nil
}

// UNSAFE
//
// Determine if the ring buffer has any data written to it or not.
//
func (r *Ring) empty() bool {
	return r.len() == 0
}

// UNSAFE
//
// Determine how many free bytes the ring buffer has.
//
func (r *Ring) freeBytes() uintptr {
	return r.size - r.len()
}

// UNSAFE
//
// Determine how many bytes have been written to the ring buffer.
//
func (r *Ring) len() uintptr {
	switch {
	// If the head is past the tail, we have used all the data from the head
	// to Size, then from 0 to Tail
	case r.head > r.tail:
		return (r.size - r.head) + r.tail

	// If the tail is past the head, we have used all the data from the head
	// to the tail
	case r.head < r.tail:
		return r.tail - r.head

	// r.head == r.tail
	default:
		return 0
	}
}

// vim: foldmethod=marker
