package diskring

import (
	"fmt"
	"io"
	"unsafe"
)

// Read up to len(buf) bytes from the buffer. This will return the number of
// bytes read, as well as any errors that happened during the read.
//
// EOF will be returned if there is no more data left in the ring buffer. An
// EOF is temporary so long as new writes are coming in.
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
		return 0, io.EOF
	}

	length := *(*uintptr)(unsafe.Pointer(&r.buf[r.head]))

	if len(buf) < int(length) {
		return 0, fmt.Errorf("buffer isn't large enough to hold chunk")
	}

	m := copy(buf, r.buf[r.head+uintptrSize:r.head+uintptrSize+length])
	return m, r.advanceHead()
}
