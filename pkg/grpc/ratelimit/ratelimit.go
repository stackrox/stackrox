package ratelimit

import (
	"github.com/stackrox/rox/pkg/httputil"
	"google.golang.org/grpc"
)

// RateLimiter is an interface that defines function for obtaining
// UnaryServer and HTTP interceptors used for rate limiting in a
// gRPC or HTTP server.
type RateLimiter interface {
	// GetUnaryServerInterceptor returns a gRPC UnaryServerInterceptor
	// that can be used to apply rate limiting.
	GetUnaryServerInterceptor() grpc.UnaryServerInterceptor

	// GetStreamServerInterceptor returns a gRPC StreamServerInterceptor
	// that can be used to apply rate limiting for gRPC streams.
	GetStreamServerInterceptor() grpc.StreamServerInterceptor

	// GetHTTPInterceptor returns an HTTPInterceptor that can be used
	// to apply rate limiting to HTTP handlers.
	GetHTTPInterceptor() httputil.HTTPInterceptor
}
