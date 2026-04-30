package relay

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errTestPermanent = errors.New("permanent failure")

func TestRunWithRetry(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		operationErrors            []error
		cancelOnFirstStreamAttempt bool
		backOffFactory             func() backoff.BackOff
		expectedErr                error
		expectedOperationAttempts  int32
	}{
		"should succeed on first try without retrying": {
			operationErrors: []error{nil},
			backOffFactory: func() backoff.BackOff {
				return backoff.NewConstantBackOff(0)
			},
			expectedOperationAttempts: 1,
		},
		"should retry once then succeed": {
			operationErrors: []error{errors.New("vsock not available"), nil},
			backOffFactory: func() backoff.BackOff {
				return backoff.NewConstantBackOff(0)
			},
			expectedOperationAttempts: 2,
		},
		"should retry through multiple consecutive failures": {
			operationErrors: []error{
				errors.New("vsock not available"),
				errors.New("vsock not available"),
				errors.New("vsock not available"),
				nil,
			},
			backOffFactory: func() backoff.BackOff {
				return backoff.NewConstantBackOff(0)
			},
			expectedOperationAttempts: 4,
		},
		"should stop without retrying when operation returns context canceled": {
			operationErrors: []error{context.Canceled},
			backOffFactory: func() backoff.BackOff {
				return backoff.NewConstantBackOff(0)
			},
			expectedErr:               context.Canceled,
			expectedOperationAttempts: 1,
		},
		"should stop without retrying when operation returns context deadline exceeded": {
			operationErrors: []error{context.DeadlineExceeded},
			backOffFactory: func() backoff.BackOff {
				return backoff.NewConstantBackOff(0)
			},
			expectedErr:               context.DeadlineExceeded,
			expectedOperationAttempts: 1,
		},
		"should stop without retrying when operation returns wrapped context canceled": {
			operationErrors: []error{fmt.Errorf("relay shut down: %w", context.Canceled)},
			backOffFactory: func() backoff.BackOff {
				return backoff.NewConstantBackOff(0)
			},
			expectedErr:               context.Canceled,
			expectedOperationAttempts: 1,
		},
		// This test cancels the context inside the operation, then relies on
		// backoff.WithContext to detect the cancelled context and return Stop
		// from NextBackOff — skipping the 1-hour configured sleep. The backoff
		// library's RetryNotify then returns ctx.Err() from the wrapped backoff.
		"should stop retries when context is canceled during backoff sleep": {
			operationErrors:            []error{errors.New("vsock not available")},
			cancelOnFirstStreamAttempt: true,
			backOffFactory: func() backoff.BackOff {
				return backoff.NewConstantBackOff(time.Hour)
			},
			expectedErr:               context.Canceled,
			expectedOperationAttempts: 1,
		},
		"should stop when backoff is exhausted": {
			operationErrors: []error{errTestPermanent},
			backOffFactory: func() backoff.BackOff {
				return &backoff.StopBackOff{}
			},
			expectedErr:               errTestPermanent,
			expectedOperationAttempts: 1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			attempts := atomic.Int32{}

			op := func(_ context.Context, _ sensor.VirtualMachineIndexReportServiceClient) error {
				attempt := attempts.Add(1)
				if tc.cancelOnFirstStreamAttempt && attempt == 1 {
					cancel()
				}
				if int(attempt) <= len(tc.operationErrors) {
					return tc.operationErrors[int(attempt)-1]
				}
				return nil
			}

			err := RunWithRetry(ctx, nil,
				WithOperation(op),
				WithBackoff(tc.backOffFactory),
			)
			if tc.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.expectedErr)
			}
			assert.Equal(t, tc.expectedOperationAttempts, attempts.Load())
		})
	}
}
