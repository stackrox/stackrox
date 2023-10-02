package ratelimit

import (
	"fmt"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/ratelimit"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
)

type rateLimiter struct {
	tokenBucketLimiter *rate.Limiter
}

// Limit implements "ratelimit.Limiter" interface.
func (limiter *rateLimiter) Limit() bool {
	return !limiter.tokenBucketLimiter.Allow()
}

// NewRateLimiter defines API rate limiter for gRPC and REST requests.
// Note: Please be aware that we're currently employing a basic token bucket
// rate limiting approach. Once the limit is reached, any additional requests
// will be declined. It's worth noting that a more effective solution would
// involve implementing request throttling before reaching the hard limit.
// However, this alternative would introduce a 1ms delay for each request
// and necessitate the creation of timers for every throttled request.
func NewRateLimiter() *rateLimiter {
	apiRequestLimitPerSec := env.CentralApiRateLimitPerSecond.IntegerSetting()
	if apiRequestLimitPerSec < 0 {
		panic(fmt.Sprintf("Negative number is not allowed for API request rate limit. Check env variable: %q", env.CentralApiRateLimitPerSecond.EnvVar()))
	}

	limit := rate.Inf
	if apiRequestLimitPerSec > 0 {
		limit = rate.Every(time.Second / time.Duration(apiRequestLimitPerSec))
	}

	// When no limit is set, we use "rate.Inf," and burst is disregarded.
	limiter := &rateLimiter{
		tokenBucketLimiter: rate.NewLimiter(limit, apiRequestLimitPerSec),
	}

	return limiter
}

// GetUnaryServerInterceptor returns a gRPC UnaryServerInterceptor.
func (limiter *rateLimiter) GetUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return ratelimit.UnaryServerInterceptor(limiter)
}

// GetHTTPInterceptor returns a HTTPInterceptor.
func (limiter *rateLimiter) GetHTTPInterceptor() httputil.HTTPInterceptor {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limiter.Limit() {
				http.Error(w, fmt.Sprintf("API call on %q is rejected by rate limiter, please retry later.", r.URL.Path), http.StatusTooManyRequests)
				return
			}

			handler.ServeHTTP(w, r)
		})
	}
}
