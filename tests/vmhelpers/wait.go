package vmhelpers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Shared poll-loop logging constants used by both VM and guest SSH wait helpers.
const (
	waitLogEveryAttempts = 5
	waitDetailMaxLen     = 300
)

// estimateMaxPollAttempts estimates how many poll iterations fit before ctx's deadline given interval.
func estimateMaxPollAttempts(ctx context.Context, interval time.Duration) (max int, known bool) {
	deadline, ok := ctx.Deadline()
	if !ok || interval <= 0 {
		return 0, false
	}
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return 1, true
	}
	return int(remaining/interval) + 1, true
}

// shouldLogWaitAttempt limits poll-loop log noise: first attempt plus every Nth.
func shouldLogWaitAttempt(attempt int) bool {
	return attempt <= 1 || attempt%waitLogEveryAttempts == 0
}

// truncateWaitDetail shortens per-attempt detail strings for log lines.
func truncateWaitDetail(detail string) string {
	detail = strings.TrimSpace(detail)
	if len(detail) <= waitDetailMaxLen {
		return detail
	}
	return detail[:waitDetailMaxLen] + fmt.Sprintf(" ... (truncated from %d bytes)", len(detail))
}

// logWaitAttempt emits one structured poll line when shouldLogWaitAttempt allows it.
func logWaitAttempt(t testing.TB, desc string, attempt, max int, maxKnown bool, detail string) {
	t.Helper()
	if !shouldLogWaitAttempt(attempt) {
		return
	}
	if maxKnown {
		left := max - attempt
		if left < 0 {
			left = 0
		}
		t.Logf("%s: attempt %d/%d (retries left: %d): %s", desc, attempt, max, left, detail)
		return
	}
	t.Logf("%s: attempt %d: %s", desc, attempt, detail)
}

// ErrAuthenticationExpired is returned when an API call fails with an
// authentication/authorization error that typically indicates an expired
// kubeconfig token or revoked credentials. Tests should stop immediately
// when this is encountered rather than retrying.
var ErrAuthenticationExpired = errors.New("authentication expired — kubeconfig token or API credentials may have expired; remaining operations will fail")

// IsAuthenticationExpired reports whether err looks like an expired or revoked
// credential. It checks gRPC Unauthenticated status codes and Kubernetes
// "Unauthorized" API errors.
func IsAuthenticationExpired(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrAuthenticationExpired) {
		return true
	}
	if s, ok := status.FromError(err); ok && s.Code() == codes.Unauthenticated {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "Unauthorized") || strings.Contains(msg, "the server has asked for the client to provide credentials")
}

// WaitOptions configures a single-condition poll loop used by Central wait helpers.
type WaitOptions struct {
	Timeout      time.Duration
	PollInterval time.Duration
	// Logf, when set, is called on each unsuccessful poll with the condition
	// description and current detail so operators can follow progress in real time.
	Logf func(string, ...any)
}

// validateWaitOptions returns an error if Timeout or PollInterval are non-positive.
func validateWaitOptions(desc string, opts WaitOptions) error {
	if opts.Timeout <= 0 {
		return fmt.Errorf("vmhelpers: %s: WaitOptions.Timeout must be positive", desc)
	}
	if opts.PollInterval <= 0 {
		return fmt.Errorf("vmhelpers: %s: WaitOptions.PollInterval must be positive", desc)
	}
	return nil
}

// pollUntil runs poll until it returns done==true or ctx deadline/opts.Timeout elapses.
// detail is included in timeout errors for targeted diagnostics.
func pollUntil(ctx context.Context, opts WaitOptions, desc string, poll func(ctx context.Context) (done bool, detail string, err error)) error {
	if err := validateWaitOptions(desc, opts); err != nil {
		return err
	}
	deadline := time.Now().Add(opts.Timeout)
	var lastDetail string
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("vmhelpers: %s: %w", desc, err)
		}
		done, detail, err := poll(ctx)
		if detail != "" {
			lastDetail = detail
		}
		if err != nil {
			if IsAuthenticationExpired(err) {
				return fmt.Errorf("vmhelpers: %s: %w: %v", desc, ErrAuthenticationExpired, err)
			}
			return fmt.Errorf("vmhelpers: %s: %w", desc, err)
		}
		if done {
			if opts.Logf != nil && detail != "" {
				opts.Logf("poll %s: done (%s)", desc, detail)
			}
			return nil
		}
		if opts.Logf != nil && detail != "" {
			opts.Logf("poll %s: waiting (%s)", desc, detail)
		}
		if time.Now().After(deadline) {
			if lastDetail != "" {
				return fmt.Errorf("vmhelpers: timeout waiting for %s after %v (last detail: %s)", desc, opts.Timeout, lastDetail)
			}
			return fmt.Errorf("vmhelpers: timeout waiting for %s after %v", desc, opts.Timeout)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("vmhelpers: %s: %w", desc, ctx.Err())
		case <-time.After(opts.PollInterval):
		}
	}
}
