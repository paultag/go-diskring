package diskring

import (
	"fmt"
	"os"
	"sync"
	"syscall"
)

type Ring struct {
	ringBase uintptr
	ringOne  uintptr
	ringTwo  uintptr

	buf []byte

	size uintptr
	head uintptr
	tail uintptr

	blockWrites bool
	mutex       sync.Mutex
}

// Create a new Ring Buffer using the underlying file (`fd`) to read and
// write entries to. Ensure that the file was opened r/w and the user is able
// to mmap the file.
func New(fd *os.File) (*Ring, error) {
	stat, err := fd.Stat()
	if err != nil {
		return nil, err
	}
	size := uintptr(stat.Size())

	if int(size)%syscall.Getpagesize() != 0 {
		return nil, fmt.Errorf("File must be aligned to page size")
	}

	// First, we need to mmap a chunk that's twice the size of the file that
	// we'll mmap, so that we can mmap two fixed offset blocks inside that
	// block.
	ringBase, err := mmap(0, size<<1,
		syscall.PROT_NONE,
		syscall.MAP_ANONYMOUS|syscall.MAP_PRIVATE,
		-1, 0)
	if err != nil {
		return nil, err
	}

	ringOne, err := mmap(ringBase, size,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_FIXED|syscall.MAP_SHARED, int(fd.Fd()), 0)
	if err != nil {
		return nil, err
	}

	if ringBase != ringOne {
		return nil, fmt.Errorf("mmap split our MAP_FIXED call")
	}

	ringTwo, err := mmap(ringBase+size, size,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_FIXED|syscall.MAP_SHARED, int(fd.Fd()), 0)
	if err != nil {
		return nil, err
	}
	if ringTwo != ringOne+size {
		return nil, fmt.Errorf("mmap split our mirror MAP_FIXED call")
	}

	return &Ring{
		size: size,
		head: 0,
		tail: 0,

		ringBase: ringBase,
		ringOne:  ringOne,
		ringTwo:  ringTwo,

		buf: *asByteSlice(ringBase, int(size<<1)),

		mutex:       sync.Mutex{},
		blockWrites: false,
	}, nil
}
