package requestinterceptor

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

func TestRequestInterceptor_noHandlers(t *testing.T) {
	ri := NewRequestInterceptor()

	// gRPC unary: should not panic, should pass through.
	interceptor := ri.UnaryServerInterceptor()
	resp, err := interceptor(context.Background(), "req",
		&grpc.UnaryServerInfo{FullMethod: "/test"},
		func(ctx context.Context, req any) (any, error) {
			return "ok", nil
		})
	assert.Equal(t, "ok", resp)
	assert.NoError(t, err)
}

func TestRequestInterceptor_dispatchesOnce(t *testing.T) {
	ri := NewRequestInterceptor()

	var count1, count2 atomic.Int32
	var captured1, captured2 *RequestParams

	ri.Add("handler1", func(rp *RequestParams) {
		count1.Add(1)
		captured1 = rp
	})
	ri.Add("handler2", func(rp *RequestParams) {
		count2.Add(1)
		captured2 = rp
	})

	// Set up a gRPC context with requestinfo so getGRPCRequestDetails works.
	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: &net.UnixAddr{Net: "pipe"}})
	req, _ := http.NewRequest(http.MethodGet, "/test/path", nil)
	rih := requestinfo.NewRequestInfoHandler()
	md := rih.AnnotateMD(ctx, req)
	ctx, riErr := rih.UpdateContextForGRPC(metadata.NewIncomingContext(ctx, md))
	require.NoError(t, riErr)

	interceptor := ri.UnaryServerInterceptor()
	_, err := interceptor(ctx, "req",
		&grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"},
		func(ctx context.Context, req any) (any, error) {
			return "ok", nil
		})
	assert.NoError(t, err)

	// Both handlers called exactly once.
	assert.Equal(t, int32(1), count1.Load())
	assert.Equal(t, int32(1), count2.Load())

	// Both received the same RequestParams pointer.
	assert.Same(t, captured1, captured2)
	assert.Equal(t, "/test/path", captured1.Path)
}

func TestRequestInterceptor_remove(t *testing.T) {
	ri := NewRequestInterceptor()

	var called atomic.Bool
	ri.Add("temp", func(rp *RequestParams) {
		called.Store(true)
	})
	ri.Remove("temp")

	// With no handlers, dispatch should be skipped entirely.
	assert.False(t, ri.hasHandlers())
}

type fakeStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (f fakeStream) Context() context.Context { return f.ctx }

func TestRequestInterceptor_Stream(t *testing.T) {
	ri := NewRequestInterceptor()

	var captured *RequestParams
	ri.Add("test", func(rp *RequestParams) {
		captured = rp
	})

	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: &net.UnixAddr{Net: "pipe"}})
	req, _ := http.NewRequest(http.MethodGet, "/stream/path", nil)
	rih := requestinfo.NewRequestInfoHandler()
	md := rih.AnnotateMD(ctx, req)
	ctx, riErr := rih.UpdateContextForGRPC(metadata.NewIncomingContext(ctx, md))
	require.NoError(t, riErr)

	interceptor := ri.StreamServerInterceptor()
	err := interceptor(nil, fakeStream{ctx: ctx},
		&grpc.StreamServerInfo{FullMethod: "/test.Service/StreamMethod"},
		func(srv any, ss grpc.ServerStream) error {
			return nil
		})
	assert.NoError(t, err)

	require.NotNil(t, captured)
	assert.Equal(t, "/stream/path", captured.Path)
	assert.Nil(t, captured.GRPCReq)
}

func TestRequestInterceptor_HTTP(t *testing.T) {
	ri := NewRequestInterceptor()

	var captured *RequestParams
	ri.Add("test", func(rp *RequestParams) {
		captured = rp
	})

	handler := ri.HTTPInterceptor()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/resource", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.NotNil(t, captured)
	assert.Equal(t, "/api/resource", captured.Path)
	assert.Equal(t, http.MethodPost, captured.Method)
	assert.Equal(t, http.StatusCreated, captured.Code)
}
