package ioutils

import (
	"context"
	"errors"
	"io"

	"github.com/stackrox/rox/pkg/sliceutils"
)

var (
	// ErrChanClosed indicates that a write failed because the channel was closed
	ErrChanClosed = errors.New("target channel was closed")
	// ErrAlreadyClosed indicates that a close operation failed because the writer had already been closed.
	ErrAlreadyClosed = errors.New("channel writer was already closed")
)

type chanWriter struct {
	ctx context.Context
	ch  chan<- []byte

	err error
}

// NewChanWriter returns a WriteCloser that writes to the given channel. It is the sole responsibility of the returned
// WriteCloser to close the channel.
func NewChanWriter(ctx context.Context, ch chan<- []byte) io.WriteCloser {
	return &chanWriter{
		ctx: ctx,
		ch:  ch,
	}
}

func (w *chanWriter) Write(buf []byte) (int, error) {
	if w.ch == nil && w.err == nil {
		w.err = ErrChanClosed
	}
	if w.err != nil {
		return 0, w.err
	}

	if len(buf) == 0 {
		return 0, nil
	}

	select {
	case w.ch <- sliceutils.ShallowClone(buf):
		return len(buf), nil
	case <-w.ctx.Done():
		w.err = w.ctx.Err()
	}
	return 0, w.err
}

func (w *chanWriter) Close() error {
	if w.ch == nil {
		return ErrAlreadyClosed
	}
	close(w.ch)
	w.ch = nil
	return nil
}
