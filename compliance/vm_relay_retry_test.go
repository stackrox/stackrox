package compliance

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stretchr/testify/require"
)

func TestRunVMRelayWithRetry(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		operationErrors            []error
		cancelOnFirstStreamAttempt bool
		backOffFactory             func() backoff.BackOff
		expectedErr                error
		expectedOperationAttempts  int32
	}{
		"should retry when operation initially fails": {
			operationErrors: []error{errors.New("vsock not available"), nil},
			backOffFactory: func() backoff.BackOff {
				return backoff.NewConstantBackOff(0)
			},
			expectedOperationAttempts: 2,
		},
		"should retry when relay run fails": {
			operationErrors: []error{errors.New("relay failed"), nil},
			backOffFactory: func() backoff.BackOff {
				return backoff.NewConstantBackOff(0)
			},
			expectedOperationAttempts: 2,
		},
		"should stop without retrying when relay run returns context canceled": {
			operationErrors: []error{context.Canceled},
			backOffFactory: func() backoff.BackOff {
				return backoff.NewConstantBackOff(0)
			},
			expectedErr:               context.Canceled,
			expectedOperationAttempts: 1,
		},
		"should stop without retrying when relay run returns context deadline exceeded": {
			operationErrors: []error{context.DeadlineExceeded},
			backOffFactory: func() backoff.BackOff {
				return backoff.NewConstantBackOff(0)
			},
			expectedErr:               context.DeadlineExceeded,
			expectedOperationAttempts: 1,
		},
		"should stop retries when context is canceled during stream creation failures": {
			operationErrors:            []error{errors.New("vsock not available")},
			cancelOnFirstStreamAttempt: true,
			backOffFactory: func() backoff.BackOff {
				return backoff.NewConstantBackOff(time.Hour)
			},
			expectedErr:               context.Canceled,
			expectedOperationAttempts: 1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			attempts := atomic.Int32{}

			c := &Compliance{
				vmRelayOperation: func(context.Context, sensor.VirtualMachineIndexReportServiceClient) error {
					attempt := attempts.Add(1)
					if tc.cancelOnFirstStreamAttempt && attempt == 1 {
						cancel()
					}
					if int(attempt) <= len(tc.operationErrors) {
						return tc.operationErrors[int(attempt)-1]
					}
					return nil
				},
				vmRelayBackOffFactory: tc.backOffFactory,
			}

			err := c.runVMRelayWithRetry(ctx, nil)
			if tc.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.ErrorIs(t, err, tc.expectedErr)
			}
			require.Equal(t, tc.expectedOperationAttempts, attempts.Load())
		})
	}
}
