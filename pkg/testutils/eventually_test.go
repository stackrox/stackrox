package testutils

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEventually_ImmediateSuccess(t *testing.T) {
	start := time.Now()
	result := Eventually(t, func() bool {
		return true
	}, 1*time.Second, 10*time.Millisecond)
	elapsed := time.Since(start)

	assert.True(t, result, "Should return true when condition is immediately true")
	assert.Less(t, elapsed, 50*time.Millisecond, "Should return immediately without waiting")
}

func TestEventually_EventualSuccess(t *testing.T) {
	var counter atomic.Int32

	result := Eventually(t, func() bool {
		count := counter.Add(1)
		return count >= 3
	}, 1*time.Second, 10*time.Millisecond)

	assert.True(t, result, "Should return true when condition eventually becomes true")
	assert.GreaterOrEqual(t, counter.Load(), int32(3), "Should have checked condition at least 3 times")
}

func TestEventually_Timeout(t *testing.T) {
	var counter atomic.Int32
	start := time.Now()

	result := Eventually(t, func() bool {
		counter.Add(1)
		return false
	}, 50*time.Millisecond, 10*time.Millisecond)
	elapsed := time.Since(start)

	assert.False(t, result, "Should return false when condition never becomes true")
	assert.GreaterOrEqual(t, elapsed, 50*time.Millisecond, "Should wait for the full timeout")
	assert.Greater(t, counter.Load(), int32(1), "Should have checked condition multiple times")
}

func TestEventually_VeryShortTimeout(t *testing.T) {
	result := Eventually(t, func() bool {
		return false
	}, 1*time.Millisecond, 1*time.Millisecond)

	assert.False(t, result, "Should handle very short timeouts")
}

func TestEventually_MultipleRetries(t *testing.T) {
	// Simulates using Eventually in a retry loop (its intended use case)
	attempts := 0
	maxAttempts := 3

	for attempts < maxAttempts {
		attempts++
		var counter atomic.Int32

		result := Eventually(t, func() bool {
			count := counter.Add(1)
			// Fail on first two attempts, succeed on third
			return attempts >= 3 && count >= 2
		}, 100*time.Millisecond, 10*time.Millisecond)

		if result {
			break
		}
	}

	assert.Equal(t, 3, attempts, "Should succeed on third attempt")
}
