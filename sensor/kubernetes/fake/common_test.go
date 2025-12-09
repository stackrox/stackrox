package fake

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestJitteredInterval(t *testing.T) {
	tests := map[string]struct {
		interval      time.Duration
		jitterPercent float64
	}{
		"60s with 5% jitter": {
			interval:      60 * time.Second,
			jitterPercent: 0.05,
		},
		"1m with 20% jitter": {
			interval:      time.Minute,
			jitterPercent: 0.20,
		},
		"100ms with 10% jitter": {
			interval:      100 * time.Millisecond,
			jitterPercent: 0.10,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			minExpected := time.Duration(float64(tt.interval) * (1 - tt.jitterPercent))
			maxExpected := time.Duration(float64(tt.interval) * (1 + tt.jitterPercent))

			// Run multiple times to verify randomness stays within bounds
			for range 100 {
				result := jitteredInterval(tt.interval, tt.jitterPercent)
				assert.GreaterOrEqual(t, result, minExpected, "jittered interval below minimum")
				assert.LessOrEqual(t, result, maxExpected, "jittered interval above maximum")
			}
		})
	}
}

func TestJitteredInterval_ZeroJitter(t *testing.T) {
	interval := 60 * time.Second
	result := jitteredInterval(interval, 0)
	assert.Equal(t, interval, result, "zero jitter should return exact interval")
}
