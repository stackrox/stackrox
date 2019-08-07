package ioutils

import (
	"io"
	"sync/atomic"
)

type countingReader struct {
	reader io.Reader
	count  *int64
}

// NewCountingReader wraps the given reader in a reader that ensures the given count variable is atomically updated
// whenever data is read.
func NewCountingReader(reader io.Reader, count *int64) io.ReadCloser {
	return &countingReader{
		reader: reader,
		count:  count,
	}
}

func (r *countingReader) Close() error {
	return Close(r.reader)
}

func (r *countingReader) Read(buf []byte) (int, error) {
	n, err := r.reader.Read(buf)
	atomic.AddInt64(r.count, int64(n))
	return n, err
}
