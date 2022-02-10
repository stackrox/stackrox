package ioutils

import (
	"io"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
)

var (
	// ErrInterrupted indicates that a read operation was interrupted.
	ErrInterrupted = errors.New("interrupted")
)

// InterruptFunc is the function performing an interruption on a reader.
type InterruptFunc func()

type interruptibleReader struct {
	io.ReadCloser
	interrupted concurrency.Flag
}

// NewInterruptibleReader returns a new reader that can be interrupted. Note that this does NOT allow interrupting an
// ongoing read operation in real time (this would in the general way only be possible with a major performance hit),
// but it allows to atomically fully stop ongoing and future read operations in the sense that they will return no data.
// This can be used in conjunction with underlying readers which are affected by a context, to ensure that canceling the
// respective context does not result in partial, possibly corrupted data being returned.
func NewInterruptibleReader(r io.Reader) (io.ReadCloser, InterruptFunc) {
	rc, _ := r.(io.ReadCloser)
	if rc == nil {
		rc = io.NopCloser(r)
	}

	ir := &interruptibleReader{
		ReadCloser: rc,
	}

	return ir, ir.interrupt
}

func (r *interruptibleReader) interrupt() {
	r.interrupted.Set(true)
}

func (r *interruptibleReader) Read(buf []byte) (int, error) {
	if r.interrupted.Get() {
		return 0, ErrInterrupted
	}

	n, err := r.ReadCloser.Read(buf)
	if r.interrupted.Get() {
		n = 0
		if err == nil {
			err = ErrInterrupted
		}
	}

	return n, err
}
