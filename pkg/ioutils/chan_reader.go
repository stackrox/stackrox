package ioutils

import (
	"context"
	"io"
)

type chanReader struct {
	ctx context.Context
	ch  <-chan []byte

	buf []byte
	err error
}

// NewChanReader returns a reader that reads chunks of bytes from a channel.
func NewChanReader(ctx context.Context, ch <-chan []byte) io.Reader {
	return &chanReader{
		ctx: ctx,
		ch:  ch,
	}
}

func (r *chanReader) Read(buf []byte) (int, error) {
	if len(buf) == 0 || r.err != nil {
		return 0, r.err
	}

	for len(r.buf) == 0 && r.err == nil {
		select {
		case chunk, ok := <-r.ch:
			if !ok {
				r.err = io.EOF
			}
			r.buf = chunk
		case <-r.ctx.Done():
			r.err = r.ctx.Err()
		}
	}

	nRead := copy(buf, r.buf)
	r.buf = r.buf[nRead:]

	return nRead, r.err
}
