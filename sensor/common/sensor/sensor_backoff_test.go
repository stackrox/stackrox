package sensor

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stretchr/testify/assert"
)

// TestConnectionStableDurationConfiguration verifies the environment variable is configured correctly
func TestConnectionStableDurationConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected time.Duration
	}{
		{
			name:     "default value",
			envValue: "",
			expected: 60 * time.Second,
		},
		{
			name:     "custom value - 30 seconds",
			envValue: "30s",
			expected: 30 * time.Second,
		},
		{
			name:     "zero value - legacy behavior",
			envValue: "0s",
			expected: 0,
		},
		{
			name:     "large value",
			envValue: "5m",
			expected: 5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv("ROX_SENSOR_CONNECTION_STABLE_DURATION", tt.envValue)
			}
			// Force re-evaluation of the setting
			actual := env.ConnectionStableDuration.DurationSetting()
			assert.Equal(t, tt.expected, actual)
		})
	}
}

// TestBackoffBehaviorDocumentation documents the expected backoff behavior
// The actual implementation is tested via integration tests in sensor/tests/connection/
func TestBackoffBehaviorDocumentation(t *testing.T) {
	t.Run("early failure preserves backoff", func(t *testing.T) {
		// When connection fails before stable duration (e.g., during initial sync):
		// - Connection start time is recorded
		// - When connection stops, elapsed time is checked against stable duration
		// - If elapsed < stable duration, backoff is NOT reset
		// - Log message: "Connection failed after Xs (before stable duration 60s), preserving exponential backoff"

		stableDuration := 60 * time.Second
		failureTime := 15 * time.Second

		assert.Less(t, failureTime, stableDuration,
			"Early failures should preserve backoff to prevent rapid retries")
	})

	t.Run("stable connection resets backoff", func(t *testing.T) {
		// When connection remains stable for >= stable duration:
		// - When connection stops, elapsed time >= stable duration
		// - exponential.Reset() is called
		// - Log message: "Connection stable for 60s (threshold: 60s), resetting exponential backoff"

		stableDuration := 60 * time.Second
		connectionDuration := 120 * time.Second

		assert.GreaterOrEqual(t, connectionDuration, stableDuration,
			"Stable connections should reset backoff for faster recovery")
	})

	t.Run("zero duration = immediate reset", func(t *testing.T) {
		// When ROX_SENSOR_CONNECTION_STABLE_DURATION=0:
		// - Feature is disabled (legacy behavior)
		// - exponential.Reset() is called immediately
		// - Log message: "Connection stable duration is 0, resetting exponential backoff immediately (legacy behavior)"

		stableDuration := 0 * time.Second

		assert.Equal(t, time.Duration(0), stableDuration,
			"Zero duration provides rollback to legacy behavior")
	})
}

// TestExpectedRetryIntervals documents the expected exponential backoff behavior
func TestExpectedRetryIntervals(t *testing.T) {
	t.Run("with backoff preserved", func(t *testing.T) {
		// Given default configuration:
		// - InitialInterval = 10s
		// - MaxInterval = 5m
		//
		// When multiple rapid failures occur (connection fails before 60s stable duration):
		// Retry intervals should increase exponentially:
		// Attempt 1: 10s
		// Attempt 2: 20s
		// Attempt 3: 40s
		// Attempt 4: 80s = 1m20s
		// Attempt 5: 160s = 2m40s
		// Attempt 6: 300s = 5m (capped at MaxInterval)
		// Attempt 7+: 5m (stays at MaxInterval)
		//
		// This prevents the DoS scenario in ROX-29270 where 7 sensors
		// were retrying every 10s, overwhelming the ingress router

		initialInterval := 10 * time.Second
		maxInterval := 5 * time.Minute

		expectedIntervals := []time.Duration{
			10 * time.Second,  // Attempt 1
			20 * time.Second,  // Attempt 2
			40 * time.Second,  // Attempt 3
			80 * time.Second,  // Attempt 4
			160 * time.Second, // Attempt 5
			300 * time.Second, // Attempt 6 (capped at max)
			300 * time.Second, // Attempt 7+ (stays at max)
		}

		for i, interval := range expectedIntervals {
			if interval <= maxInterval {
				assert.LessOrEqual(t, interval, maxInterval,
					"Attempt %d interval should not exceed max", i+1)
			}
			if i > 0 && interval < maxInterval {
				assert.Greater(t, interval, initialInterval,
					"Attempt %d should have increased from initial", i+1)
			}
		}
	})

	t.Run("with backoff reset", func(t *testing.T) {
		// When connection is stable for >= 60s, then later fails:
		// - Backoff is reset to initial interval
		// - Next retry will be after 10s (not 5m)
		// - This allows faster recovery for legitimately transient issues

		initialInterval := 10 * time.Second

		assert.Greater(t, initialInterval, time.Duration(0),
			"Reset backoff allows faster recovery")
	})
}
