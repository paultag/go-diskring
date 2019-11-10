package diskring

import (
	"fmt"
	"io"
	"unsafe"
)

func (r *Ring) Read(buf []byte) (int, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.len() == 0 {
		return 0, io.EOF
	}

	length := *(*uintptr)(unsafe.Pointer(&r.buf[r.head]))

	if len(buf) < int(length) {
		return 0, fmt.Errorf("buffer isn't large enough to hold chunk")
	}

	m := copy(buf, r.buf[r.head+uintptrSize:r.head+uintptrSize+length])
	return m, r.advanceHead()
}
