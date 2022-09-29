package pgutils

import (
	"time"

	"github.com/stackrox/rox/pkg/timeutil"
)

const (
	interval = 5 * time.Second
	timeout  = 5 * time.Minute
)

// RetryExecQuery is used to specify how long to retry to successfully run an exec query
// that fails with Transient errors
func RetryExecQuery(fn func() error) error {
	// Shape fn to match the RetryQuery below
	fnWithReturn := func() (struct{}, error) {
		return struct{}{}, fn()
	}
	_, err := RetryQuery(fnWithReturn)
	return err
}

// RetryQuery is used to specify how long to retry to successfully run a query
// that fails with Transient errors
func RetryQuery[T any](fn func() (T, error)) (T, error) {
	// Run query immediately
	if val, err := fn(); err == nil || !isTransientError(err) {
		return val, err
	}

	expirationTimer := time.NewTimer(timeout)
	defer timeutil.StopTimer(expirationTimer)

	intervalTicker := time.NewTicker(interval)
	defer intervalTicker.Stop()

	var err error
	for {
		select {
		case <-expirationTimer.C:
			log.Fatalf("unsuccessful in reconnecting to the database: %v. Exiting...", err)
		case <-intervalTicker.C:
			// Uses err outside the for loop to allow for the expiration to show the last err received
			// and provide context for the expiration
			var ret T
			ret, err = fn()
			if err == nil || !isTransientError(err) {
				return ret, err
			}
		}
	}
}
