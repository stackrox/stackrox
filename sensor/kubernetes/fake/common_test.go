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
		minExpected   time.Duration
		maxExpected   time.Duration
	}{
		"60s with 5% jitter": {
			interval:      60 * time.Second,
			jitterPercent: 0.05,
			minExpected:   57 * time.Second,
			maxExpected:   63 * time.Second,
		},
		"1m with 20% jitter": {
			interval:      time.Minute,
			jitterPercent: 0.20,
			minExpected:   48 * time.Second,
			maxExpected:   72 * time.Second,
		},
		"100ms with 10% jitter": {
			interval:      100 * time.Millisecond,
			jitterPercent: 0.10,
			minExpected:   90 * time.Millisecond,
			maxExpected:   110 * time.Millisecond,
		},
		"100ms with 0% jitter": {
			interval:      100 * time.Millisecond,
			jitterPercent: 0.00,
			minExpected:   100 * time.Millisecond,
			maxExpected:   100 * time.Millisecond,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Run multiple times to verify randomness stays within bounds
			for range 100 {
				result := jitteredInterval(tt.interval, tt.jitterPercent)
				assert.GreaterOrEqual(t, result, tt.minExpected, "jittered interval below minimum")
				assert.LessOrEqual(t, result, tt.maxExpected, "jittered interval above maximum")
			}
		})
	}
}
