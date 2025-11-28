package sensor

import (
	"testing"
	"time"

	"github.com/cenkalti/backoff/v3"
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
			envValue: "60s",
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
			// Always set env var explicitly for deterministic test behavior
			t.Setenv("ROX_SENSOR_CONNECTION_STABLE_DURATION", tt.envValue)

			actual := env.ConnectionStableDuration.DurationSetting()
			assert.Equal(t, tt.expected, actual)
		})
	}
}

// TestShouldResetBackoff tests the backoff reset decision logic
func TestShouldResetBackoff(t *testing.T) {
	tests := []struct {
		name            string
		connectionStart time.Time
		stableDuration  time.Duration
		expectedReset   bool
	}{
		{
			name:            "legacy mode - zero duration always resets",
			connectionStart: time.Now().Add(-30 * time.Second),
			stableDuration:  0,
			expectedReset:   true,
		},
		{
			name:            "connection stable - should reset",
			connectionStart: time.Now().Add(-70 * time.Second),
			stableDuration:  60 * time.Second,
			expectedReset:   true,
		},
		{
			name:            "early failure - preserve backoff",
			connectionStart: time.Now().Add(-15 * time.Second),
			stableDuration:  60 * time.Second,
			expectedReset:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldResetBackoff(tt.connectionStart, tt.stableDuration)
			assert.Equal(t, tt.expectedReset, result)
		})
	}
}

// TestExponentialBackoffProgression tests actual exponential backoff behavior
func TestExponentialBackoffProgression(t *testing.T) {
	exponential := backoff.NewExponentialBackOff()
	exponential.InitialInterval = 10 * time.Second
	exponential.MaxInterval = 5 * time.Minute
	exponential.MaxElapsedTime = 0
	exponential.RandomizationFactor = 0 // Disable randomization for deterministic testing
	exponential.Multiplier = 2           // Explicit multiplier for doubling
	exponential.Reset()                  // Reset to initialize state

	intervals := []time.Duration{}
	for i := 0; i < 7; i++ {
		intervals = append(intervals, exponential.NextBackOff())
	}

	// Verify exponential growth
	assert.Equal(t, 10*time.Second, intervals[0], "First interval should be InitialInterval")
	assert.Greater(t, intervals[1], intervals[0], "Second interval should be greater than first")
	assert.Greater(t, intervals[2], intervals[1], "Third interval should be greater than second")

	// Verify capping at MaxInterval
	assert.LessOrEqual(t, intervals[6], 5*time.Minute, "All intervals should be capped at MaxInterval")
}
