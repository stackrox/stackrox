package ratelimit

import (
	"math"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func TestNewRateLimiterUnlimited(t *testing.T) {
	rl := newRateLimiter(-1, 0)
	assert.Equal(t, rate.Inf, rl.tokenBucketLimiter.Limit())

	rl = newRateLimiter(0, 0)
	assert.Equal(t, rate.Inf, rl.tokenBucketLimiter.Limit())
}

func TestNewRateLimiterLimitedValue(t *testing.T) {
	expectedRatePerSec := 100
	rl := newRateLimiter(expectedRatePerSec, 0)

	assert.Equal(t, rate.Limit(expectedRatePerSec), rl.tokenBucketLimiter.Limit())
	assert.Equal(t, expectedRatePerSec, rl.tokenBucketLimiter.Burst())
}

func TestIncreaseLimit(t *testing.T) {
	noLimitRL := newRateLimiter(0, 0)
	assert.Equal(t, rate.Inf, noLimitRL.tokenBucketLimiter.Limit())

	noLimitRL.IncreaseLimit(1)
	assert.Equal(t, rate.Inf, noLimitRL.tokenBucketLimiter.Limit())

	expectedRatePerSec := 100
	rl := newRateLimiter(expectedRatePerSec, 0)
	assert.Equal(t, rate.Limit(expectedRatePerSec), rl.tokenBucketLimiter.Limit())
	assert.Equal(t, expectedRatePerSec, rl.tokenBucketLimiter.Burst())

	for _, limitDelta := range []int{-10, 0, math.MaxInt} {
		rl.IncreaseLimit(limitDelta)
		assert.Equal(t, rate.Limit(expectedRatePerSec), rl.tokenBucketLimiter.Limit())
		assert.Equal(t, expectedRatePerSec, rl.tokenBucketLimiter.Burst())
	}

	rl.IncreaseLimit(10)
	expectedRatePerSec += 10
	assert.Equal(t, rate.Every(time.Second/time.Duration(expectedRatePerSec)), rl.tokenBucketLimiter.Limit())
	assert.Equal(t, expectedRatePerSec, rl.tokenBucketLimiter.Burst())
}

func TestDecreaseLimit(t *testing.T) {
	noLimitRL := newRateLimiter(0, 0)
	assert.Equal(t, rate.Inf, noLimitRL.tokenBucketLimiter.Limit())

	noLimitRL.DecreaseLimit(1)
	assert.Equal(t, rate.Inf, noLimitRL.tokenBucketLimiter.Limit())

	expectedRatePerSec := 100
	rl := newRateLimiter(expectedRatePerSec, 0)
	assert.Equal(t, rate.Limit(expectedRatePerSec), rl.tokenBucketLimiter.Limit())
	assert.Equal(t, expectedRatePerSec, rl.tokenBucketLimiter.Burst())

	for _, limitDelta := range []int{-10, 0, math.MaxInt, expectedRatePerSec} {
		rl.DecreaseLimit(limitDelta)
		assert.Equal(t, rate.Limit(expectedRatePerSec), rl.tokenBucketLimiter.Limit())
		assert.Equal(t, expectedRatePerSec, rl.tokenBucketLimiter.Burst())
	}

	rl.DecreaseLimit(10)
	expectedRatePerSec -= 10
	assert.Equal(t, rate.Every(time.Second/time.Duration(expectedRatePerSec)), rl.tokenBucketLimiter.Limit())
	assert.Equal(t, expectedRatePerSec, rl.tokenBucketLimiter.Burst())
}

func TestLimitNoThrottle(t *testing.T) {
	tests := []struct {
		name                string
		maxPerSec           int
		maxThrottleDuration time.Duration
	}{
		{"NoLimit 0s", 0, 0},
		{"WithLimitHit 0s", 1, 0},
		{"NoLimit 1ns", 0, time.Nanosecond},
		{"WithLimitHit 1ns", 1, time.Nanosecond},
		{"NoLimit under 1s", 0, time.Second - 1},
		{"WithLimitHit under 1s", 1, time.Second - 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := newRateLimiter(tt.maxPerSec, tt.maxThrottleDuration)

			r1 := rl.Limit()
			r2 := rl.Limit()

			assert.False(t, r1)
			assert.True(t, tt.maxPerSec == 0 || r2)
			assert.False(t, tt.maxPerSec == 0 && r2)
		})
	}
}

func TestLimitWithThrottle(t *testing.T) {
	tests := []struct {
		name                string
		maxPerSec           int
		maxThrottleDuration time.Duration
	}{
		{"NoLimit 1s", 0, time.Second},
		{"WithLimitHit 1s", 1, time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := newRateLimiter(tt.maxPerSec, tt.maxThrottleDuration)

			var wg sync.WaitGroup

			numCalls := tt.maxPerSec + 10
			resultChan := make(chan bool, numCalls)

			for i := 0; i < numCalls; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					resultChan <- rl.Limit()
				}()
			}

			go func() {
				wg.Wait()
				close(resultChan)
			}()

			countLimitHit := 0
			for result := range resultChan {
				if result {
					countLimitHit++
				}
			}

			if tt.maxPerSec == 0 {
				assert.Equal(t, countLimitHit, 0)
			} else {
				assert.Less(t, countLimitHit, numCalls)
				assert.GreaterOrEqual(t, countLimitHit, tt.maxPerSec)
			}
		})
	}
}

func BenchmarkRateLimiter(b *testing.B) {
	tests := []struct {
		name                string
		maxPerSec           int
		maxThrottleDuration time.Duration
	}{
		{"NoLimit NoThrottle", 0, 0},
		{"WithLimitHit NoThrottle", 1, 0},
		{"WithLimitNoHit NoThrottle", math.MaxInt - 1, 0},
		{"NoLimit Throttle 1s", 0, time.Second},
		{"WithLimitHit Throttle 1s", 1, time.Second},
		{"WithLimitNoHit Throttle 1s", math.MaxInt - 1, time.Second},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			l := NewRateLimiter(tt.maxPerSec, tt.maxThrottleDuration)
			for i := 0; i < b.N; i++ {
				l.Limit()
			}
		})
	}
}
