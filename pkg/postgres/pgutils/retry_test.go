package pgutils

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// transientErr creates a transient PostgreSQL error that will trigger retries
func transientErr() error {
	return &pgconn.PgError{
		Code: "08006", // connection_failure - a transient error code
	}
}

// nonTransientErr creates a non-transient error that should not trigger retries
func nonTransientErr() error {
	return errors.New("non-transient error")
}

func TestRetry_SuccessOnFirstAttempt(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	err := Retry(ctx, func() error {
		attempts++
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, attempts, "should succeed on first attempt")
}

func TestRetry_NonTransientErrorFailsFast(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	err := Retry(ctx, func() error {
		attempts++
		return nonTransientErr()
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "non-retryable error")
	assert.Equal(t, 1, attempts, "should not retry non-transient errors")
}

func TestRetry_TransientErrorRetries(t *testing.T) {
	// Set shorter intervals for faster testing
	require.NoError(t, os.Setenv("ROX_POSTGRES_QUERY_RETRY_INTERVAL", "100ms"))
	require.NoError(t, os.Setenv("ROX_POSTGRES_QUERY_RETRY_TIMEOUT", "1s"))

	ctx := context.Background()
	attempts := 0
	maxAttempts := 3

	err := Retry(ctx, func() error {
		attempts++
		if attempts < maxAttempts {
			return transientErr()
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, maxAttempts, attempts, "should retry until success")

	require.NoError(t, os.Unsetenv("ROX_POSTGRES_QUERY_RETRY_INTERVAL"))
	require.NoError(t, os.Unsetenv("ROX_POSTGRES_QUERY_RETRY_TIMEOUT"))
}

func TestRetry_TransientErrorTimeout(t *testing.T) {
	// Set very short timeout for faster testing
	require.NoError(t, os.Setenv("ROX_POSTGRES_QUERY_RETRY_INTERVAL", "50ms"))
	require.NoError(t, os.Setenv("ROX_POSTGRES_QUERY_RETRY_TIMEOUT", "200ms"))

	ctx := context.Background()
	attempts := 0

	start := time.Now()
	err := Retry(ctx, func() error {
		attempts++
		return transientErr()
	})
	elapsed := time.Since(start)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "retry timer is expired")
	assert.Greater(t, attempts, 1, "should retry at least once before timeout")
	assert.GreaterOrEqual(t, elapsed, 200*time.Millisecond, "should respect timeout")

	require.NoError(t, os.Unsetenv("ROX_POSTGRES_QUERY_RETRY_INTERVAL"))
	require.NoError(t, os.Unsetenv("ROX_POSTGRES_QUERY_RETRY_TIMEOUT"))
}

func TestRetry_DisabledRetriesFailFast(t *testing.T) {
	// Enable disable flag
	require.NoError(t, os.Setenv("ROX_POSTGRES_DISABLE_QUERY_RETRIES", "true"))

	ctx := context.Background()
	attempts := 0

	start := time.Now()
	err := Retry(ctx, func() error {
		attempts++
		return transientErr()
	})
	elapsed := time.Since(start)

	assert.Error(t, err)
	assert.Equal(t, 1, attempts, "should only attempt once when retries disabled")
	assert.Less(t, elapsed, 100*time.Millisecond, "should fail fast without waiting")

	require.NoError(t, os.Unsetenv("ROX_POSTGRES_DISABLE_QUERY_RETRIES"))
}

func TestRetry_ContextCancellation(t *testing.T) {
	// Set shorter intervals for faster testing
	require.NoError(t, os.Setenv("ROX_POSTGRES_QUERY_RETRY_INTERVAL", "100ms"))
	require.NoError(t, os.Setenv("ROX_POSTGRES_QUERY_RETRY_TIMEOUT", "5s"))

	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0

	// Cancel context after 2 attempts
	go func() {
		time.Sleep(150 * time.Millisecond)
		cancel()
	}()

	err := Retry(ctx, func() error {
		attempts++
		return transientErr()
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "retry context is done")

	require.NoError(t, os.Unsetenv("ROX_POSTGRES_QUERY_RETRY_INTERVAL"))
	require.NoError(t, os.Unsetenv("ROX_POSTGRES_QUERY_RETRY_TIMEOUT"))
}

func TestRetryIfPostgres_DelegatesToRetry(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	err := RetryIfPostgres(ctx, func() error {
		attempts++
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, attempts)
}

func TestRetry_CustomIntervalIsRespected(t *testing.T) {
	// Set custom interval
	require.NoError(t, os.Setenv("ROX_POSTGRES_QUERY_RETRY_INTERVAL", "200ms"))

	ctx := context.Background()
	attempts := 0
	var attemptTimes []time.Time

	err := Retry(ctx, func() error {
		attempts++
		attemptTimes = append(attemptTimes, time.Now())
		if attempts < 3 {
			return transientErr()
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)

	// Check that intervals between attempts are roughly 200ms
	if len(attemptTimes) >= 2 {
		interval1 := attemptTimes[1].Sub(attemptTimes[0])
		// Allow some tolerance for timing (150-250ms)
		assert.Greater(t, interval1, 150*time.Millisecond)
		assert.Less(t, interval1, 300*time.Millisecond)
	}

	require.NoError(t, os.Unsetenv("ROX_POSTGRES_QUERY_RETRY_INTERVAL"))
	require.NoError(t, os.Unsetenv("ROX_POSTGRES_QUERY_RETRY_TIMEOUT"))
}
