//go:build test

package vmhelpers

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// exitError returns an *exec.ExitError with the given exit code.
func exitError(t *testing.T, code int) *exec.ExitError {
	t.Helper()
	err := exec.Command("sh", "-c", fmt.Sprintf("exit %d", code)).Run()
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, code, exitErr.ExitCode())
	return exitErr
}

func TestClassifySSHFailure_Smoke(t *testing.T) {
	t.Parallel()

	exit255 := exitError(t, 255)
	exit1 := exitError(t, 1)

	tests := map[string]struct {
		stderr    string
		err       error
		wantSSH   bool
		wantRetry bool
		wantCat   string
	}{
		"banner timeout is retryable ssh transport": {
			stderr: "Connection timed out during banner exchange", err: exit255,
			wantSSH: true, wantRetry: true, wantCat: "banner-timeout",
		},
		"auth failure is terminal ssh transport": {
			stderr: "Permission denied (publickey)", err: exit255,
			wantSSH: true, wantRetry: false, wantCat: "authentication",
		},
		"remote command failure is not ssh transport": {
			stderr: "metadata already locked by 3814", err: exit1,
			wantSSH: false, wantRetry: false, wantCat: "",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			gotSSH, gotRetry, gotCat := classifySSHFailure(tc.stderr, tc.err)
			require.Equal(t, tc.wantSSH, gotSSH, "isSSH")
			require.Equal(t, tc.wantRetry, gotRetry, "retryable")
			require.Equal(t, tc.wantCat, gotCat, "category")
		})
	}
}

func TestSSHReachabilityPolicy_ClassifyFailureStopsEarlyOnAuthThreshold(t *testing.T) {
	t.Parallel()

	policy := DefaultSSHReachabilityPolicy
	counters := &sshProbeCounters{authFailures: policy.AuthFailureThreshold - 1}
	decision := policy.classifyFailure(
		counters,
		Virtctl{Username: "cloud-user"},
		errors.New("exit status 255"),
		"Permission denied (publickey,gssapi-keyex,gssapi-with-mic).",
	)

	require.Error(t, decision.terminalErr)
	require.ErrorIs(t, decision.terminalErr, ErrSSHAuthenticationFailed)
	require.Contains(t, decision.detail, "ssh auth not accepted")
}

func TestFormatGuestCommandOutputForError(t *testing.T) {
	t.Parallel()
	require.Equal(t, "<no guest stdout/stderr>", formatGuestCommandOutputForError(" \n\t "))
	require.Equal(t, "status output", formatGuestCommandOutputForError("status output"))

	long := strings.Repeat("x", guestCommandErrorMaxLen+50)
	got := formatGuestCommandOutputForError(long)
	require.Contains(t, got, "(truncated from")
	require.True(t, strings.HasPrefix(got, strings.Repeat("x", guestCommandErrorMaxLen)))
}
