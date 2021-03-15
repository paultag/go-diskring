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

// Read up to len(buf) bytes from the buffer. This will return the number of
// bytes read, as well as any errors that happened during the read.
//
// If the buffer can't hold the entirety of the record, this function will
// error out. Be sure that the largest entry in the buffer can fit in the
// provided `buf`, or it will forever cycle trying to read that one entry.
//
// After the data is copied to the buf, the ring buffer head will be advanced.
//
//
func (r *Ring) Read(buf []byte) (int, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.len() == 0 {
		r.mutex.Unlock()
		<-r.wakeup
		r.mutex.Lock()
	}

	length := *(*uintptr)(unsafe.Pointer(&r.buf[r.cursor.head]))

	if len(buf) < int(length) {
		return 0, fmt.Errorf("buffer isn't large enough to hold chunk")
	}

	m := copy(buf, r.buf[r.cursor.head+uintptrSize:r.cursor.head+uintptrSize+length])
	return m, r.advanceHead()
}

// vim: foldmethod=marker
