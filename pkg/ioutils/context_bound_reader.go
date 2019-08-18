package ioutils

import (
	"context"
	"io"
	"io/ioutil"
)

type contextBoundReader struct {
	io.ReadCloser
	ctx context.Context

	readReqC  chan []byte
	readRespC chan readResp
}

type readResp struct {
	n   int
	err error
}

// NewContextBoundReader returns a reader where every call to `Read` is bounded by the given context. I.e., whenever
// the context is canceled or exceeds its deadline, `Read` will return immediately with a context error.
// Note: Close is not affected and is passed through to the underlying Reader's Close method, if any.
func NewContextBoundReader(ctx context.Context, reader io.Reader) io.ReadCloser {
	rc, _ := reader.(io.ReadCloser)
	if rc == nil {
		rc = ioutil.NopCloser(reader)
	}
	cbr := &contextBoundReader{
		ReadCloser: rc,
		ctx:        ctx,
		readReqC:   make(chan []byte),
		readRespC:  make(chan readResp),
	}
	go cbr.readLoop()
	return cbr
}

func (r *contextBoundReader) Read(buf []byte) (int, error) {
	if r.ctx.Err() != nil {
		return 0, r.ctx.Err()
	}

	select {
	case r.readReqC <- buf:
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	}

	select {
	case readResp := <-r.readRespC:
		return readResp.n, readResp.err
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	}
}

func (r *contextBoundReader) readLoop() {
	for {
		select {
		case <-r.ctx.Done():
			return
		case buf := <-r.readReqC:
			n, err := r.ReadCloser.Read(buf)
			select {
			case <-r.ctx.Done():
				return
			case r.readRespC <- readResp{n: n, err: err}:
			}
		}
	}
}
