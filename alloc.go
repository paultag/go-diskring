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
