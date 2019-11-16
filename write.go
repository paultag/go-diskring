package diskring

import (
	"fmt"
	"unsafe"
)

// Prevent any new writes from hitting the Ring. This will hang all writes,
// but it will allow a process to read the buffer completely, then unlock the
// buffer if taking new writes is absolutely unacceptable.
func (r *Ring) BlockWrites() {
	r.mutex.Lock()
	r.blockWrites = true
	r.mutex.Unlock()
}

// Allow writes to the buffer after calling `BlockWrites`.
func (r *Ring) UnblockWrites() {
	r.mutex.Lock()
	r.blockWrites = false
	r.mutex.Unlock()
}

// Write a block of data into the disk ring. If there's not enough data in the
// diskring, this will advance the head until we can fit the data in. If the
// data is more than 1/4 the size of the ring, the write will fail because
// it's an arbitrary number I picked.
func (r *Ring) Write(buf []byte) error {
	if len(buf) > int(r.size/4) {
		return fmt.Errorf("data is too large")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	blen := uintptr(len(buf))
	for {
		if (blen + uintptrSize) > r.freeBytes() {
			if err := r.advanceHead(); err != nil {
				return err
			}
			continue
		}
		break
	}

	m := copy(r.buf[r.tail+uintptrSize:], buf)
	*(*uintptr)(unsafe.Pointer(&r.buf[r.tail])) = uintptr(m)
	r.tail = ((r.tail + uintptrSize + uintptr(m)) % r.size)

	return nil
}
