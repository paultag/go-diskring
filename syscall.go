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
	"syscall"
	"unsafe"
)

// *facepalm*
//
// syscall.Mmap won't let us to the hackery we need. This will let us map a
// slice twice the size of the file, then do two fixed maps inside that
// map.
//
// very gross much wow
func mmap(addr uintptr, length uintptr, prot int, flags int, fd int, offset int64) (uintptr, error) {
	r0, _, e1 := syscall.Syscall6(syscall.SYS_MMAP, addr, length,
		uintptr(prot), uintptr(flags), uintptr(fd), uintptr(offset))
	xaddr := uintptr(r0)
	if e1 != 0 {
		return 0, fmt.Errorf("errno: %d", e1)
	}
	return xaddr, nil
}

// syscall.Munmap won't let us unmap on a uintptr since it works in terms of
// (a very sensible!) []byte abstraction. This will let us unmap a specific
// address, due to how we create our []byte abstraction.
func munmap(addr uintptr, length uintptr) error {
	_, _, e1 := syscall.Syscall(syscall.SYS_MUNMAP, addr, length, 0)
	if e1 != 0 {
		return fmt.Errorf("errno: %d", e1)
	}
	return nil
}

// just.... just don't look at me.
//
// this is maybe the unsafest thing I've done in go. turn a pointer (provided
// as a uint) into a go byte slice D:
func asByteSlice(base uintptr, size int) *[]byte {
	var b = struct {
		addr uintptr
		len  int
		cap  int
	}{base, size, size}
	return (*[]byte)(unsafe.Pointer(&b))
}

// vim: foldmethod=marker
