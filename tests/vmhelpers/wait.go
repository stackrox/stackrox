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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
// credential. It checks gRPC Unauthenticated status codes and Kubernetes API
// errors classified as Unauthorized by k8s API machinery.
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
	return apierrors.IsUnauthorized(err)
}
