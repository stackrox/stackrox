package ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/ratelimit"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/sync"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
)

type rateLimiter struct {
	mutex sync.Mutex

	maxThrottleDuration time.Duration
	tokenBucketLimiter  *rate.Limiter
}

func newRateLimiter(maxPerSec int, maxThrottleDuration time.Duration) *rateLimiter {
	limit := rate.Inf
	if maxPerSec > 0 {
		limit = rate.Every(time.Second / time.Duration(maxPerSec))
	}

	// When no limit is set, we use "rate.Inf," and burst is disregarded.
	limiter := &rateLimiter{
		maxThrottleDuration: maxThrottleDuration,
		tokenBucketLimiter:  rate.NewLimiter(limit, maxPerSec),
	}

	return limiter
}

// Limit implements "ratelimit.Limiter" interface.
func (limiter *rateLimiter) Limit() bool {
	if limiter.maxThrottleDuration < time.Second {
		return !limiter.tokenBucketLimiter.Allow()
	}

	ctx, cancelFnc := context.WithTimeout(context.Background(), limiter.maxThrottleDuration)
	defer cancelFnc()

	return limiter.tokenBucketLimiter.Wait(ctx) != nil
}

func (limiter *rateLimiter) modifyRateLimit(limitDelta int) {
	if limiter.tokenBucketLimiter.Limit() == rate.Inf {
		return
	}

	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()

	newBurst := limiter.tokenBucketLimiter.Burst() + limitDelta
	if 0 < newBurst {
		limiter.tokenBucketLimiter.SetBurst(newBurst)
		limiter.tokenBucketLimiter.SetLimit(rate.Every(time.Second / time.Duration(newBurst)))
	}
}

func (limiter *rateLimiter) IncreaseLimit(limitDelta int) {
	if limitDelta <= 0 {
		return
	}

	limiter.modifyRateLimit(limitDelta)
}

func (limiter *rateLimiter) DecreaseLimit(limitDelta int) {
	if limitDelta <= 0 {
		return
	}

	limiter.modifyRateLimit(-limitDelta)
}

// GetUnaryServerInterceptor returns a gRPC UnaryServerInterceptor.
func (limiter *rateLimiter) GetUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return ratelimit.UnaryServerInterceptor(limiter)
}

// GetStreamServerInterceptor returns a gRPC StreamServerInterceptor.
func (limiter *rateLimiter) GetStreamServerInterceptor() grpc.StreamServerInterceptor {
	return ratelimit.StreamServerInterceptor(limiter)
}

// GetHTTPInterceptor returns a HTTPInterceptor.
func (limiter *rateLimiter) GetHTTPInterceptor() httputil.HTTPInterceptor {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limiter.Limit() {
				msg := fmt.Sprintf("APIRateLimiter call on %q is rejected by rate limiter, please retry later.", r.URL.Path)
				http.Error(w, msg, http.StatusTooManyRequests)

				return
			}

			handler.ServeHTTP(w, r)
		})
	}
}

// NewRateLimiter defines rate limiter any type of events. The rate limit
// will be considered unlimited when the value of maxPerSec is less than or
// equal to zero. In the event that the rate limit is exceeded, we will
// initiate request throttling. If incoming requests are not processed within
// the allocated throttle time, they will begin to time out, resulting in
// an error response. Any maxThrottleDuration less than 1 second will not be
// taken into account.
func NewRateLimiter(maxPerSec int, maxThrottleDuration time.Duration) RateLimiter {
	return newRateLimiter(maxPerSec, maxThrottleDuration)
}
