package diskring

import (
	"fmt"
	"syscall"
	"unsafe"
)

//
func mmap(addr uintptr, length uintptr, prot int, flags int, fd int, offset int64) (uintptr, error) {
	r0, _, e1 := syscall.Syscall6(syscall.SYS_MMAP, addr, length,
		uintptr(prot), uintptr(flags), uintptr(fd), uintptr(offset))
	xaddr := uintptr(r0)
	if e1 != 0 {
		return 0, fmt.Errorf("errno: %d", e1)
	}
	return xaddr, nil
}

//
func asByteSlice(base uintptr, size int) *[]byte {
	var b = struct {
		addr uintptr
		len  int
		cap  int
	}{base, size, size}
	return (*[]byte)(unsafe.Pointer(&b))
}
