package ratelimit

import (
	"github.com/stackrox/rox/pkg/httputil"
	"google.golang.org/grpc"
)

// RateLimiter is an interface that defines function for obtaining
// UnaryServer and HTTP interceptors used for rate limiting in a
// gRPC or HTTP server.
type RateLimiter interface {
	// Limit returns true when the event should be rejected.
	Limit() bool

	// LimitNoThrottle returns true when the event should be rejected.
	// It returns without throttling the function call.
	LimitNoThrottle() bool

	// IncreaseLimit increases the allowed rate of events. If rate limiter
	// is unlimited, no change is made. The argument 'limitDelta' has to be
	// bigger than 0, otherwise no change is made.
	IncreaseLimit(limitDelta int)

	// DecreaseLimit decreases the allowed rate of events. If rate limiter
	// is unlimited, no change is made. The argument 'limitDelta' has to be
	// bigger than 0, otherwise no change is made.
	DecreaseLimit(limitDelta int)

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
