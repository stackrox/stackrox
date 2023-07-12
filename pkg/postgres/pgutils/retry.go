package pgutils

import (
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	interval = 5 * time.Second
	timeout  = 5 * time.Minute
)

// Retry is used to specify how long to retry to successfully run a query with 1 return value
// that fails with transient errors
func Retry(fn func() error) error {
	// Shape fn to match the Retry2 below
	fnWithReturn := func() (struct{}, error) {
		return struct{}{}, fn()
	}
	_, err := Retry2(fnWithReturn)
	return err
}

// Retry2 is used to specify how long to retry to successfully run a query with 2 return values
// that fails with transient errors
func Retry2[T any](fn func() (T, error)) (T, error) {
	// Shape fn to match the Retry3 below
	fnWithReturn := func() (T, struct{}, error) {
		val, err := fn()
		return val, struct{}{}, err
	}
	val, _, err := Retry3(fnWithReturn)
	return val, err
}

// RetryIfPostgres is same as Retry except that it retries iff postgres is enabled.
// that fails with transient errors
func RetryIfPostgres(fn func() error) error {
	return Retry(fn)
}

// Retry3 is used to specify how long to retry to successfully run a query with 3 return values
// that fails with transient errors
func Retry3[T any, U any](fn func() (T, U, error)) (T, U, error) {
	// Run query immediately
	if val1, val2, err := fn(); err == nil || !IsTransientError(err) {
		if err != nil && err != pgx.ErrNoRows {
			log.Debugf("UNEXPECTED: found permanent error: %+v", err)
		}
		return val1, val2, err
	}

	expirationTimer := time.NewTimer(timeout)
	defer expirationTimer.Stop()

	intervalTicker := time.NewTicker(interval)
	defer intervalTicker.Stop()

	var err error
	var ret1 T
	var ret2 U
	for {
		select {
		case <-expirationTimer.C:
			return ret1, ret2, err
		case <-intervalTicker.C:
			// Uses err outside the for loop to allow for the expiration to show the last err received
			// and provide context for the expiration
			ret1, ret2, err = fn()
			if err == nil || !IsTransientError(err) {
				if err != nil && err != pgx.ErrNoRows {
					log.Debugf("UNEXPECTED: found permanent error: %+v", err)
				}
				return ret1, ret2, err
			}
		}
	}
}
