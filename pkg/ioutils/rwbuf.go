package ioutils

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
)

// RWBuf is a buffer that acts as a combination of a `bytes.Buffer` and a `bytes.Reader`, additionally supporting
// file-backed storage to avoid consuming too much memory.
type RWBuf struct {
	memBuf bytes.Buffer
	opts   RWBufOptions

	err  error
	size int64

	tmpFile *os.File
}

// RWBufOptions controls the behavior of an RWBuf.
type RWBufOptions struct {
	MemLimit  int // Maximum amount of data to store in memory. -1 means no memory storage, 0 means unlimited.
	HardLimit int // Maximum amount of data to store regardless of medium. <= 0 means no limit.
}

// NewRWBuf returns a buffer that can be used for reading and writing. The buffer is memory-backed up to a given size,
// and then switches to being file-backed.
// The object itself serves as an `io.WriteCloser`. Contents can be accessed via a call to the `Contents()` method.
// This method must be called before a call to `Close()`; after `Close()` has been invoked, the buffer contents might no
// longer be accessible.
func NewRWBuf(opts RWBufOptions) *RWBuf {
	return &RWBuf{
		opts: opts,
	}
}

// Contents returns a ReaderAt for reading the contents, along with the total size of the buffer, or an error, if any
// of the preceding write operations yielded an error.
func (b *RWBuf) Contents() (io.ReaderAt, int64, error) {
	if b.err != nil {
		return nil, 0, b.err
	}
	if b.tmpFile != nil {
		return b.tmpFile, b.size, nil
	}
	return bytes.NewReader(b.memBuf.Bytes()), b.size, nil
}

// Write implements io.Writer.
func (b *RWBuf) Write(buf []byte) (int, error) {
	if b.err != nil {
		return 0, b.err
	}
	if len(buf) == 0 {
		return 0, nil
	}

	n, err := b.doWrite(buf)
	b.size += int64(n)
	b.err = err
	return n, err
}

func (b *RWBuf) doWrite(buf []byte) (int, error) {
	if b.opts.HardLimit > 0 && b.size+int64(len(buf)) > int64(b.opts.HardLimit) {
		return 0, errors.Errorf("write would exceed hard limit of %d bytes", b.opts.HardLimit)
	}

	if b.tmpFile == nil {
		if b.opts.MemLimit == 0 || b.memBuf.Len()+len(buf) <= b.opts.MemLimit {
			return b.memBuf.Write(buf)
		}

		var err error
		b.tmpFile, err = ioutil.TempFile("", "rwbuf")
		if err != nil {
			return 0, errors.Wrap(err, "creating temporary file")
		}
		if _, err := io.Copy(b.tmpFile, &b.memBuf); err != nil {
			return 0, errors.Wrap(err, "writing out file contents")
		}
	}

	return b.tmpFile.Write(buf)
}

// Close implements io.WriteCloser.
func (b *RWBuf) Close() error {
	if b.tmpFile != nil {
		name := b.tmpFile.Name()
		_ = b.tmpFile.Close()
		_ = os.Remove(name)
	}
	return nil
}
