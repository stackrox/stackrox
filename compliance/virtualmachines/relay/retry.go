package relay

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/metrics"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/sender"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/stream"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
)

const retryRateLimitKey = "vm-relay-retry"

// Operation creates and runs the VM relay pipeline (stream + sender + relay).
type Operation func(ctx context.Context, sensorClient sensor.VirtualMachineIndexReportServiceClient) error

type retryConfig struct {
	operation      Operation
	backOffFactory func() backoff.BackOff
}

// RetryOption configures RunWithRetry behavior.
type RetryOption func(*retryConfig)

// WithOperation overrides the default relay operation (primarily for testing).
func WithOperation(op Operation) RetryOption {
	return func(c *retryConfig) {
		c.operation = op
	}
}

// WithBackoff overrides the default backoff factory (primarily for testing).
func WithBackoff(factory func() backoff.BackOff) RetryOption {
	return func(c *retryConfig) {
		c.backOffFactory = factory
	}
}

// RunWithRetry runs the VM relay operation with exponential backoff.
// The primary retry target is vsock listener bind failures (e.g. KubeVirt not yet installed).
// Once the listener is up and Relay.Run enters its long-lived accept/send loop, that loop only
// exits on context cancellation, which is treated as a permanent (non-retryable) error.
//
// Retries continue indefinitely until the parent context is cancelled at shutdown.
func RunWithRetry(ctx context.Context, sensorClient sensor.VirtualMachineIndexReportServiceClient, umh UnconfirmedMessageHandler, opts ...RetryOption) error {
	cfg := retryConfig{
		backOffFactory: defaultBackOff,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.operation == nil {
		cfg.operation = makeDefaultOperation(umh)
	}

	operation := func() error {
		err := cfg.operation(ctx, sensorClient)
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return backoff.Permanent(err)
		}
		return err
	}
	notification := func(err error, _ time.Duration) {
		metrics.RetryAttempts.Inc()
		if ctx.Err() != nil {
			return
		}
		logging.GetRateLimitedLogger().WarnL(
			retryRateLimitKey,
			"VM relay failed: %v",
			err,
		)
	}
	return backoff.RetryNotify(operation, backoff.WithContext(cfg.backOffFactory(), ctx), notification)
}

// defaultBackOff returns an exponential backoff that retries indefinitely (MaxElapsedTime=0)
// until the parent context is cancelled at shutdown. The interval is capped at 5 minutes.
func defaultBackOff() backoff.BackOff {
	eb := backoff.NewExponentialBackOff()
	eb.MaxInterval = 5 * time.Minute
	eb.MaxElapsedTime = 0
	return eb
}

func makeDefaultOperation(umh UnconfirmedMessageHandler) Operation {
	return func(ctx context.Context, sensorClient sensor.VirtualMachineIndexReportServiceClient) error {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		reportStream, err := stream.New()
		if err != nil {
			return err
		}

		reportSender := sender.New(sensorClient)

		vmRelay := New(
			reportStream,
			reportSender,
			umh,
			env.VMRelayMaxReportsPerMinute.FloatSetting(),
			env.VMRelayStaleAckThreshold.DurationSetting(),
			env.VMIndexReportRelayCacheSlots.IntegerSetting(),
			env.VMIndexReportRelayCacheTTL.DurationSetting(),
		)
		return vmRelay.Run(ctx)
	}
}
