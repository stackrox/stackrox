package ioutils

import (
	"io"
	"sync"

	"github.com/pkg/errors"
)

type lazyReaderAt struct {
	reader io.Reader
	size   int64

	mutex sync.RWMutex
	buf   []byte
	err   error
}

var (
	errBufferStolen = errors.New("buffer was stolen")
)

// LazyReaderAt is a ReaderAt that reads from an underlying reader into a buffer
// in a lazy fashion.
type LazyReaderAt interface {
	io.ReaderAt
	// StealBuffer steals the underlying buffer from the reader. The length of the buffer
	// will contain complete contents from the beginning of the reader, while the remaining
	// capacity is unused.
	// After calling this function, subsequent calls to it will return nil, and
	// subsequent calls to `ReadAt` will return an error.
	// This can be used in conjunction with `NewLazyReaderAtWithBuffer` to recycle a buffer.
	StealBuffer() []byte
}

// NewLazyReaderAtWithBuffer returns a ReaderAt that lazily reads from the underlying reader.
func NewLazyReaderAtWithBuffer(reader io.Reader, size int64, buf []byte) LazyReaderAt {
	return &lazyReaderAt{
		reader: reader,
		size:   size,

		buf: buf[:0],
	}
}

func (r *lazyReaderAt) StealBuffer() []byte {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	buf := r.buf
	r.buf = nil
	r.err = errBufferStolen

	return buf
}

func (r *lazyReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if off >= r.size {
		return 0, io.EOF
	}
	n, err := r.tryReadAt(p, off)
	if n != 0 || err != nil {
		return n, err
	}

	r.readUntil(off + int64(len(p)))

	return r.tryReadAt(p, off)
}

func (r *lazyReaderAt) tryReadAt(p []byte, off int64) (int, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	pos := int64(len(r.buf))
	if off+int64(len(p)) <= pos {
		return copy(p, r.buf[off:]), nil
	}

	if r.err != nil {
		// If we are in error state, that means we have read as much as we could from the reader.
		// Read whatever is left in the buffer, accompanied by the error.
		if off >= int64(len(r.buf)) {
			return 0, r.err
		}
		return copy(p, r.buf[off:]), r.err
	}

	return 0, nil
}

func (r *lazyReaderAt) readUntil(pos int64) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if pos > r.size {
		pos = r.size
	}

	if pos <= int64(len(r.buf)) {
		return
	}

	if int64(cap(r.buf)) < pos {
		newBuf := make([]byte, len(r.buf), r.size)
		copy(newBuf, r.buf)
		r.buf = newBuf
	}

	oldPos := len(r.buf)
	r.buf = r.buf[:pos]
	nRead, err := io.ReadFull(r.reader, r.buf[oldPos:])
	r.buf = r.buf[:oldPos+nRead]
	if err != nil {
		r.err = err
	} else if pos == r.size {
		r.err = io.EOF
	}
}
