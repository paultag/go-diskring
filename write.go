package diskring

import (
	"fmt"
	"unsafe"
)

func (r *Ring) BlockWrites() {
	r.mutex.Lock()
	r.blockWrites = true
	r.mutex.Unlock()
}

func (r *Ring) UnblockWrites() {
	r.mutex.Lock()
	r.blockWrites = false
	r.mutex.Unlock()
}

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
