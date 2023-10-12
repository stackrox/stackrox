package ratelimit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func TestNewRateLimiterUnlimited(t *testing.T) {
	rl := NewRateLimiter(-1)
	assert.Equal(t, rate.Inf, rl.tokenBucketLimiter.Limit())

	rl = NewRateLimiter(0)
	assert.Equal(t, rate.Inf, rl.tokenBucketLimiter.Limit())
}

func TestNewRateLimiterLimitedValue(t *testing.T) {
	expectedRatePerSec := 100
	rl := NewRateLimiter(expectedRatePerSec)

	assert.Equal(t, rate.Limit(expectedRatePerSec), rl.tokenBucketLimiter.Limit())
	assert.Equal(t, expectedRatePerSec, rl.tokenBucketLimiter.Burst())
}

func TestIncreaseLimit(t *testing.T) {
	noLimitRL := NewRateLimiter(0)
	assert.Equal(t, rate.Inf, noLimitRL.tokenBucketLimiter.Limit())

	noLimitRL.IncreaseLimit(1)
	assert.Equal(t, rate.Inf, noLimitRL.tokenBucketLimiter.Limit())

	expectedRatePerSec := 100
	rl := NewRateLimiter(expectedRatePerSec)
	assert.Equal(t, rate.Limit(expectedRatePerSec), rl.tokenBucketLimiter.Limit())
	assert.Equal(t, expectedRatePerSec, rl.tokenBucketLimiter.Burst())

	rl.IncreaseLimit(10)
	expectedRatePerSec += 10
	assert.Equal(t, rate.Limit(expectedRatePerSec), rl.tokenBucketLimiter.Limit())
	assert.Equal(t, expectedRatePerSec, rl.tokenBucketLimiter.Burst())
}

func TestDecreaseLimit(t *testing.T) {
	noLimitRL := NewRateLimiter(0)
	assert.Equal(t, rate.Inf, noLimitRL.tokenBucketLimiter.Limit())

	noLimitRL.DecreaseLimit(1)
	assert.Equal(t, rate.Inf, noLimitRL.tokenBucketLimiter.Limit())

	expectedRatePerSec := 100
	rl := NewRateLimiter(expectedRatePerSec)
	assert.Equal(t, rate.Limit(expectedRatePerSec), rl.tokenBucketLimiter.Limit())
	assert.Equal(t, expectedRatePerSec, rl.tokenBucketLimiter.Burst())

	rl.DecreaseLimit(10)
	expectedRatePerSec -= 10
	assert.Equal(t, rate.Limit(expectedRatePerSec), rl.tokenBucketLimiter.Limit())
	assert.Equal(t, expectedRatePerSec, rl.tokenBucketLimiter.Burst())
}

func BenchmarkNoLimit(b *testing.B) {
	l := NewRateLimiter(0)
	for i := 0; i < b.N; i++ {
		l.Limit()
	}
}

func BenchmarkWithLimitHit(b *testing.B) {
	limit := b.N / 10
	if limit < 1 {
		limit = 1
	}

	l := NewRateLimiter(limit)
	for i := 0; i < b.N; i++ {
		l.Limit()
	}
}

func BenchmarkWithLimitNoHit(b *testing.B) {
	l := NewRateLimiter(b.N + 1000)
	for i := 0; i < b.N; i++ {
		l.Limit()
	}
}
