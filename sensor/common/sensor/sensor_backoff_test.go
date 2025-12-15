package sensor

import (
	"context"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/stretchr/testify/assert"
)

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

// TestHandleBackoffOnConnectionStop tests the backoff management logic on connection stop
func TestHandleBackoffOnConnectionStop(t *testing.T) {
	tests := []struct {
		name            string
		connectionStart time.Time
		stableDuration  time.Duration
		err             error
		expectReset     bool
	}{
		{
			name:            "legacy mode - always resets",
			connectionStart: time.Now().Add(-5 * time.Second),
			stableDuration:  0,
			err:             nil,
			expectReset:     true,
		},
		{
			name:            "stable connection - resets backoff",
			connectionStart: time.Now().Add(-70 * time.Second),
			stableDuration:  60 * time.Second,
			err:             nil,
			expectReset:     true,
		},
		{
			name:            "early failure - preserves backoff",
			connectionStart: time.Now().Add(-15 * time.Second),
			stableDuration:  60 * time.Second,
			err:             assert.AnError,
			expectReset:     false,
		},
		{
			name:            "early cancellation - preserves backoff",
			connectionStart: time.Now().Add(-15 * time.Second),
			stableDuration:  60 * time.Second,
			err:             context.Canceled,
			expectReset:     false,
		},
		{
			name:            "early stop without error - preserves backoff",
			connectionStart: time.Now().Add(-15 * time.Second),
			stableDuration:  60 * time.Second,
			err:             nil,
			expectReset:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exponential := backoff.NewExponentialBackOff()
			exponential.InitialInterval = 1 * time.Second
			exponential.MaxInterval = 5 * time.Minute
			exponential.MaxElapsedTime = 0
			exponential.RandomizationFactor = 0 // Disable randomization for deterministic testing
			exponential.Multiplier = 2          // Explicit multiplier for doubling
			exponential.Reset()

			// Advance backoff state to verify reset behavior
			_ = exponential.NextBackOff()
			interval2 := exponential.NextBackOff()

			wasReset := handleBackoffOnConnectionStop(exponential, tt.connectionStart, tt.stableDuration, tt.err)

			assert.Equal(t, tt.expectReset, wasReset)

			// Verify backoff state after handling
			nextInterval := exponential.NextBackOff()
			if tt.expectReset {
				// After reset, should be back to initial interval
				assert.Equal(t, exponential.InitialInterval, nextInterval,
					"After reset, next interval should be InitialInterval")
			} else {
				// Without reset, should continue exponential growth from where it was
				// Before: 1s, 2s (interval2). After should be 4s (greater than interval2)
				assert.Greater(t, nextInterval, interval2,
					"Without reset, backoff should continue growing beyond %v, got %v", interval2, nextInterval)
			}
		})
	}
}

// TestShouldDisableReconcile tests the reconciliation disable decision
func TestShouldDisableReconcile(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectDisabled bool
	}{
		{
			name:           "no error - reconciliation enabled",
			err:            nil,
			expectDisabled: false,
		},
		{
			name:           "generic error - reconciliation enabled",
			err:            assert.AnError,
			expectDisabled: false,
		},
		{
			name:           "can't reconcile error - disable reconciliation",
			err:            errCantReconcile,
			expectDisabled: true,
		},
		{
			name:           "large payload error - disable reconciliation",
			err:            errLargePayload,
			expectDisabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldDisableReconcile(tt.err)
			assert.Equal(t, tt.expectDisabled, result)
		})
	}
}

// TestHandleReconnectionError tests the reconnection error handling logic
func TestHandleReconnectionError(t *testing.T) {
	tests := []struct {
		name                   string
		err                    error
		expectDisableReconcile bool
	}{
		{
			name:                   "no error - reconciliation enabled",
			err:                    nil,
			expectDisableReconcile: false,
		},
		{
			name:                   "generic error - reconciliation enabled",
			err:                    assert.AnError,
			expectDisableReconcile: false,
		},
		{
			name:                   "can't reconcile error - disable reconciliation",
			err:                    errCantReconcile,
			expectDisableReconcile: true,
		},
		{
			name:                   "large payload error - disable reconciliation",
			err:                    errLargePayload,
			expectDisableReconcile: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldDisable := handleReconnectionError(tt.err)
			assert.Equal(t, tt.expectDisableReconcile, shouldDisable)
		})
	}
}
