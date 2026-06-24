//go:build test

package vmhelpers

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
		Timeout:      150 * time.Millisecond,
		PollInterval: 15 * time.Millisecond,
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

	err = pollUntil(ctx, WaitOptions{Timeout: 1 * time.Second, PollInterval: -1}, "bad", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "PollInterval must be positive")
}

func TestIsAuthenticationExpired(t *testing.T) {
	tests := map[string]struct {
		err  error
		want bool
	}{
		"nil error":                        {err: nil, want: false},
		"gRPC Unauthenticated":             {err: status.Error(codes.Unauthenticated, "token expired"), want: true},
		"Unauthorized substring":           {err: errors.New("request failed: Unauthorized 401"), want: true},
		"lowercase unauthorized":           {err: errors.New("unauthorized access"), want: true},
		"server asked for credentials":     {err: errors.New("the server has asked for the client to provide credentials"), want: true},
		"wrapped ErrAuthenticationExpired": {err: fmt.Errorf("op: %w", ErrAuthenticationExpired), want: true},
		"unrelated error":                  {err: errors.New("connection refused"), want: false},
		"gRPC unavailable (not auth)":      {err: status.Error(codes.Unavailable, "service down"), want: false},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, IsAuthenticationExpired(tc.err))
		})
	}
}
