package vmscraper

import (
	"errors"
	"time"

	"github.com/stackrox/rox/sensor/common/virtualmachine/vsockclient"
)

const (
	initialBackoff = 10 * time.Second
	maxBackoffCap  = 30 * time.Minute
	maxReconcile   = 5 * time.Minute
)

func backoffCap(pollInterval time.Duration) time.Duration {
	return min(pollInterval, maxBackoffCap)
}

func reconcilePeriod(pollInterval time.Duration) time.Duration {
	return min(pollInterval, maxReconcile)
}

// nextBackoff returns the next backoff after a retryable failure or NACK.
func nextBackoff(current, pollInterval time.Duration) time.Duration {
	cap := backoffCap(pollInterval)
	if current <= 0 {
		return min(initialBackoff, cap)
	}
	return min(current*2, cap)
}

// isRetryable reports whether a GetReport/dial error should grow backoff.
// UnknownMethod is permanent; everything else (including ErrInternal / NotReady /
// EOF / dial failures, and ErrBusy when that sentinel exists) is retryable.
func isRetryable(err error) bool {
	return err != nil && !errors.Is(err, vsockclient.ErrUnknownMethod)
}
