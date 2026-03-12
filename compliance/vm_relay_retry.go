package compliance

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/stream"
	"github.com/stackrox/rox/pkg/logging"
)

var vmRelayLog = logging.LoggerForModule()

// createVMRelayStreamWithRetry creates a vsock stream with retry and backoff.
// Retries until the factory succeeds or ctx is cancelled. Used to recover when vsock
// becomes available after KubeVirt is installed.
func createVMRelayStreamWithRetry(ctx context.Context, createStream func() (*stream.VsockIndexReportStream, error)) (*stream.VsockIndexReportStream, error) {
	eb := backoff.NewExponentialBackOff()
	eb.MaxInterval = 30 * time.Second
	eb.MaxElapsedTime = 0 // Retry indefinitely until vsock becomes available or ctx cancelled
	b := backoff.WithContext(eb, ctx)

	var reportStream *stream.VsockIndexReportStream
	operation := func() error {
		var err error
		reportStream, err = createStream()
		return err
	}
	notify := func(err error, d time.Duration) {
		vmRelayLog.Infof("Vsock bind failed, retrying in %0.2f seconds: %v", d.Seconds(), err)
	}
	if err := backoff.RetryNotify(operation, b, notify); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, err
	}
	return reportStream, nil
}
