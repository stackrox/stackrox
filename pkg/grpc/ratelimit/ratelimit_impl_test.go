package ratelimit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func TestNewRateLimiterNegativeLimitPanics(t *testing.T) {
	assert.Panics(t, func() { NewRateLimiter(-1) })
}

func TestNewRateLimiterZeroIsUnlimited(t *testing.T) {
	rl := NewRateLimiter(0)
	assert.Equal(t, rate.Inf, rl.tokenBucketLimiter.Limit())
	assert.Equal(t, 0, rl.tokenBucketLimiter.Burst())
}

func TestNewRateLimiterLimitedValue(t *testing.T) {
	expectedRatePerSec := 100
	rl := NewRateLimiter(expectedRatePerSec)

	assert.Equal(t, rate.Limit(expectedRatePerSec), rl.tokenBucketLimiter.Limit())
	assert.Equal(t, expectedRatePerSec, rl.tokenBucketLimiter.Burst())
}
