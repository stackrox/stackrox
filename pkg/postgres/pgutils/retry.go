package pgutils

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stackrox/rox/pkg/env"
)


// Retry is used to specify how long to retry to successfully run a query with 1 return value
// that fails with transient errors
func Retry(ctx context.Context, fn func() error) error {
	// Shape fn to match the Retry2 below
	fnWithReturn := func() (struct{}, error) {
		return struct{}{}, fn()
	}
	_, err := Retry2(ctx, fnWithReturn)
	return err
}

// Retry2 is used to specify how long to retry to successfully run a query with 2 return values
// that fails with transient errors
func Retry2[T any](ctx context.Context, fn func() (T, error)) (T, error) {
	// Shape fn to match the Retry3 below
	fnWithReturn := func() (T, struct{}, error) {
		val, err := fn()
		return val, struct{}{}, err
	}
	val, _, err := Retry3(ctx, fnWithReturn)
	return val, err
}

// RetryIfPostgres is same as Retry except that it retries iff postgres is enabled.
// that fails with transient errors
func RetryIfPostgres(ctx context.Context, fn func() error) error {
	return Retry(ctx, fn)
}

// Retry3 is used to specify how long to retry to successfully run a query with 3 return values
// that fails with transient errors
func Retry3[T any, U any](ctx context.Context, fn func() (T, U, error)) (T, U, error) {
	var ret1 T
	var ret2 U
	if err := ctx.Err(); err != nil {
		return ret1, ret2, fmt.Errorf("retry context is done: %w", err)
	}

	// Run query immediately
	val1, val2, err := fn()
	if err == nil || !IsTransientError(err) {
		if err != nil && err != pgx.ErrNoRows {
			log.Debugf("UNEXPECTED: found non-retryable error: %+v", err)
			return ret1, ret2, fmt.Errorf("found non-retryable error: %w", err)

		}
		return val1, val2, err
	}

	// If retries are disabled, fail fast after single attempt
	if env.PostgresDisableQueryRetries.BooleanSetting() {
		return ret1, ret2, err
	}

	expirationTimer := time.NewTimer(env.PostgresQueryRetryTimeout.DurationSetting())
	defer expirationTimer.Stop()

	intervalTicker := time.NewTicker(env.PostgresQueryRetryInterval.DurationSetting())
	defer intervalTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ret1, ret2, fmt.Errorf("retry context is done: %w", ctx.Err())
		case <-expirationTimer.C:
			return ret1, ret2, fmt.Errorf("retry timer is expired: %w", err)
		case <-intervalTicker.C:
			// Uses err outside the for loop to allow for the expiration to show the last err received
			// and provide context for the expiration
			ret1, ret2, err = fn()
			if err == nil || !IsTransientError(err) {
				if err != nil && err != pgx.ErrNoRows {
					log.Debugf("UNEXPECTED: found non-retryable error: %+v", err)
					return ret1, ret2, fmt.Errorf("found non-retryable error: %w", err)
				}
				return ret1, ret2, err
			}
		}
	}
}
