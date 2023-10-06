package ratelimit

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func TestNewRateLimiterNegativeLimitPanics(t *testing.T) {
	t.Setenv(env.CentralApiRateLimitPerSecond.EnvVar(), "-1")
	assert.Panics(t, func() { NewRateLimiter() })
}

func TestNewRateLimiterDefaultUnlimited(t *testing.T) {
	rl := NewRateLimiter()

	assert.Equal(t, rate.Inf, rl.tokenBucketLimiter.Limit())
	assert.Equal(t, 0, rl.tokenBucketLimiter.Burst())
}

func TestNewRateLimiterZeroIsUnlimited(t *testing.T) {
	t.Setenv(env.CentralApiRateLimitPerSecond.EnvVar(), "0")

	rl := NewRateLimiter()
	assert.Equal(t, rate.Inf, rl.tokenBucketLimiter.Limit())
	assert.Equal(t, 0, rl.tokenBucketLimiter.Burst())
}

func TestNewRateLimiterLimitedValue(t *testing.T) {
	expectedRatePerSec := 100

	t.Setenv(env.CentralApiRateLimitPerSecond.EnvVar(), fmt.Sprint(expectedRatePerSec))
	rl := NewRateLimiter()

	assert.Equal(t, rate.Limit(expectedRatePerSec), rl.tokenBucketLimiter.Limit())
	assert.Equal(t, expectedRatePerSec, rl.tokenBucketLimiter.Burst())
}
