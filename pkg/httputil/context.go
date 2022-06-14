package httputil

import (
	"context"
	"io"
	"net/http"

	"github.com/stackrox/rox/pkg/concurrency"
)

type contextBoundRoundTripper struct {
	ctx      concurrency.ErrorWaitable
	delegate http.RoundTripper
}

// ContextBoundRoundTripper returns an http.RoundTripper that delegates to the rt for performing requests, but ensures
// every request is canceled when `ctx` is canceled.
func ContextBoundRoundTripper(ctx concurrency.ErrorWaitable, rt http.RoundTripper) http.RoundTripper {
	return &contextBoundRoundTripper{
		ctx:      ctx,
		delegate: rt,
	}
}

type cancelOnCloseReader struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (r *cancelOnCloseReader) Close() error {
	r.cancel()
	return r.ReadCloser.Close()
}

func (r *contextBoundRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if ctxErr := r.ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}

	ctxBoundReq, cancel := ContextBoundRequest(r.ctx, req)

	resp, err := r.delegate.RoundTrip(ctxBoundReq)
	if err != nil {
		cancel()
		return nil, err
	}
	if resp.Body != nil {
		resp.Body = &cancelOnCloseReader{
			ReadCloser: resp.Body,
			cancel:     cancel,
		}
	} else {
		cancel()
	}

	return resp, nil
}

// ContextBoundRequest returns a new *http.Request with a context that is canceled when ctx is canceled. No values of
// ctx will be propagated to the context of the resulting request.
// Callers should ensure that the returned cancel function is called eventually, otherwise this function might leak
// Goroutines.
func ContextBoundRequest(ctx concurrency.ErrorWaitable, req *http.Request) (*http.Request, context.CancelFunc) {
	subCtx, cancel := context.WithCancel(req.Context())
	concurrency.CancelContextOnSignal(subCtx, cancel, ctx)
	return req.WithContext(subCtx), cancel
}
