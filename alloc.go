package diskring

import (
	"io"
	"unsafe"
)

var (
	uintptrSize = unsafe.Sizeof(uintptr(0))
)

// UNSAFE
func (r *Ring) advanceHead() error {
	if r.len() == 0 {
		return io.EOF
	}
	length := *(*uintptr)(unsafe.Pointer(&r.buf[r.head]))
	r.head = (r.head + length + uintptrSize) % r.size
	return nil
}

// UNSAFE
func (r *Ring) empty() bool {
	return r.len() == 0
}

// UNSAFE
func (r *Ring) freeBytes() uintptr {
	return r.size - r.len()
}

// UNSAFE
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
