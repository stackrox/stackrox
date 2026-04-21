package compliance

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/stackrox/rox/compliance/virtualmachines/relay"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/sender"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stretchr/testify/require"
)

type vmRelayRunnerFunc func(ctx context.Context) error

func (f vmRelayRunnerFunc) Run(ctx context.Context) error {
	return f(ctx)
}

type fakeIndexReportStream struct{}

func (fakeIndexReportStream) Start(context.Context) (<-chan *v1.VMReport, error) {
	return nil, nil
}

type fakeIndexReportSender struct{}

func (fakeIndexReportSender) Send(context.Context, *v1.VMReport) error {
	return nil
}

func TestRunVMRelayWithRetry(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		streamErrors               []error
		runErrors                  []error
		cancelOnFirstStreamAttempt bool
		backOffFactory             func() backoff.BackOff
		expectedErr                error
		expectedStreamAttempts     int
		expectedRunAttempts        int
	}{
		"should retry when stream creation initially fails": {
			streamErrors: []error{errors.New("vsock not available")},
			runErrors:    []error{nil},
			backOffFactory: func() backoff.BackOff {
				return backoff.NewConstantBackOff(0)
			},
			expectedStreamAttempts: 2,
			expectedRunAttempts:    1,
		},
		"should retry from stream creation when relay run fails": {
			runErrors: []error{errors.New("relay failed"), nil},
			backOffFactory: func() backoff.BackOff {
				return backoff.NewConstantBackOff(0)
			},
			expectedStreamAttempts: 2,
			expectedRunAttempts:    2,
		},
		"should stop without retrying when relay run returns context canceled": {
			runErrors: []error{context.Canceled},
			backOffFactory: func() backoff.BackOff {
				return backoff.NewConstantBackOff(0)
			},
			expectedErr:            context.Canceled,
			expectedStreamAttempts: 1,
			expectedRunAttempts:    1,
		},
		"should stop retries when context is canceled during stream creation failures": {
			streamErrors:               []error{errors.New("vsock not available")},
			cancelOnFirstStreamAttempt: true,
			backOffFactory: func() backoff.BackOff {
				return backoff.NewConstantBackOff(time.Hour)
			},
			expectedErr:            context.Canceled,
			expectedStreamAttempts: 1,
			expectedRunAttempts:    0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			streamAttempts := 0
			runAttempts := 0

			c := &Compliance{
				vmRelayStreamFactory: func() (relay.IndexReportStream, error) {
					streamAttempts++
					if tc.cancelOnFirstStreamAttempt && streamAttempts == 1 {
						cancel()
					}
					if streamAttempts <= len(tc.streamErrors) {
						if err := tc.streamErrors[streamAttempts-1]; err != nil {
							return nil, err
						}
					}
					return fakeIndexReportStream{}, nil
				},
				vmRelaySenderFactory: func(sensor.VirtualMachineIndexReportServiceClient) sender.IndexReportSender {
					return fakeIndexReportSender{}
				},
				vmRelayFactory: func(relay.IndexReportStream, sender.IndexReportSender) vmRelayRunner {
					return vmRelayRunnerFunc(func(context.Context) error {
						runAttempts++
						if runAttempts <= len(tc.runErrors) {
							return tc.runErrors[runAttempts-1]
						}
						return nil
					})
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
			require.Equal(t, tc.expectedStreamAttempts, streamAttempts)
			require.Equal(t, tc.expectedRunAttempts, runAttempts)
		})
	}
}
