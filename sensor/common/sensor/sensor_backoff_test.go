package sensor

import (
	"context"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/assert"
)

// TestHandleBackoffOnConnectionStop tests the backoff management logic on connection stop
func TestHandleBackoffOnConnectionStop(t *testing.T) {
	tests := []struct {
		name        string
		syncDone    bool
		err         error
		expectReset bool
	}{
		{
			name:        "sync completed - resets backoff",
			syncDone:    true,
			err:         nil,
			expectReset: true,
		},
		{
			name:        "sync completed with error - resets backoff",
			syncDone:    true,
			err:         assert.AnError,
			expectReset: true,
		},
		{
			name:        "sync not completed - preserves backoff",
			syncDone:    false,
			err:         assert.AnError,
			expectReset: false,
		},
		{
			name:        "sync not completed with cancellation - preserves backoff",
			syncDone:    false,
			err:         context.Canceled,
			expectReset: false,
		},
		{
			name:        "sync not completed without error - preserves backoff",
			syncDone:    false,
			err:         nil,
			expectReset: false,
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

			// Create syncDone signal
			syncDone := concurrency.NewSignal()
			if tt.syncDone {
				syncDone.Signal()
			}

			wasReset := handleBackoffOnConnectionStop(exponential, &syncDone, tt.err)

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
