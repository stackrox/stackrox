//go:build test

package vmhelpers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPollUntil_ImmediateSuccess(t *testing.T) {
	ctx := context.Background()
	err := pollUntil(ctx, WaitOptions{
		Timeout:      100 * time.Millisecond,
		PollInterval: 1 * time.Millisecond,
	}, "immediate", func(context.Context) (bool, string, error) {
		return true, "ok", nil
	})
	require.NoError(t, err)
}

func TestPollUntil_SucceedsAfterRetries(t *testing.T) {
	ctx := context.Background()
	var polls int
	err := pollUntil(ctx, WaitOptions{
		Timeout:      300 * time.Millisecond,
		PollInterval: 5 * time.Millisecond,
	}, "retry-test", func(context.Context) (bool, string, error) {
		polls++
		return polls >= 3, "stepping", nil
	})
	require.NoError(t, err)
	require.Equal(t, 3, polls)
}

func TestPollUntil_TimesOutWithDetail(t *testing.T) {
	ctx := context.Background()
	err := pollUntil(ctx, WaitOptions{
		Timeout:      30 * time.Millisecond,
		PollInterval: 5 * time.Millisecond,
	}, "timeout-test", func(context.Context) (bool, string, error) {
		return false, "still waiting", nil
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "timeout")
	require.Contains(t, err.Error(), "still waiting")
}

func TestPollUntil_RejectsInvalidOptions(t *testing.T) {
	ctx := context.Background()
	err := pollUntil(ctx, WaitOptions{Timeout: -1, PollInterval: 1 * time.Millisecond}, "bad", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Timeout must be positive")
}
