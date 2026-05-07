//go:build test

package vmhelpers

import (
	"context"
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

func TestActivationFromSubscriptionManagerOutput(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		output string
		want   bool
	}{
		"Current": {
			output: "+-------------------------------------------+\n   Overall Status: Current",
			want:   true,
		},
		"NotCurrent": {
			output: "Overall Status: Unknown",
			want:   false,
		},
		"Empty": {
			output: "",
			want:   false,
		},
		"current lowercase": {
			output: "Overall Status: current",
			want:   true,
		},
		"CURRENT uppercase": {
			output: "Overall Status: CURRENT",
			want:   true,
		},
		"registered lowercase": {
			output: "Overall Status: registered",
			want:   true,
		},
		"REGISTERED uppercase": {
			output: "Overall Status: REGISTERED",
			want:   true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, activationFromSubscriptionManagerOutput(tt.output))
		})
	}
}

func TestOverallStatusFromSubscriptionManagerOutput(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		output string
		status string
		found  bool
	}{
		"current status": {
			output: "Overall Status: Current",
			status: "Current",
			found:  true,
		},
		"not registered status with leading spaces": {
			output: "   Overall Status: Not registered",
			status: "Not registered",
			found:  true,
		},
		"case insensitive prefix": {
			output: "overall status: Unknown",
			status: "Unknown",
			found:  true,
		},
		"missing status line": {
			output: "subscription-manager output without marker",
			status: "",
			found:  false,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			status, found := overallStatusFromSubscriptionManagerOutput(tc.output)
			require.Equal(t, tc.status, status)
			require.Equal(t, tc.found, found)
		})
	}
}

func TestActivationStatusFromCommandOutput(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		stdout    string
		stderr    string
		cmdErr    error
		activated bool
		wantErr   bool
	}{
		"current status and nil error": {
			stdout:    "Overall Status: Current",
			activated: true,
			wantErr:   false,
		},
		"not registered with exit status is non fatal": {
			stdout:    "Overall Status: Not registered",
			stderr:    "exit status 1",
			cmdErr:    errors.New("exit status 1"),
			activated: false,
			wantErr:   false,
		},
		"unknown status with exit status is non fatal": {
			stdout:    "Overall Status: Unknown",
			stderr:    "exit status 1",
			cmdErr:    errors.New("exit status 1"),
			activated: false,
			wantErr:   false,
		},
		"registered status with exit status is non fatal and activated": {
			stdout:    "Overall Status: Registered",
			stderr:    "exit status 1",
			cmdErr:    errors.New("exit status 1"),
			activated: true,
			wantErr:   false,
		},
		"missing status line with exit status is fatal": {
			stdout:    "some other output",
			stderr:    "rpc failure",
			cmdErr:    errors.New("exit status 1"),
			activated: false,
			wantErr:   true,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			activated, details, err := activationStatusFromCommandOutput(tc.stdout, tc.stderr, tc.cmdErr)
			require.Equal(t, tc.activated, activated)
			if tc.wantErr {
				require.Error(t, err)
				require.NotEmpty(t, details)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, details)
		})
	}
}

func TestClassifySSHFailure(t *testing.T) {
	t.Parallel()
	exit255 := exitError(t, 255)
	exit1 := exitError(t, 1)
	exit42 := exitError(t, 42)

	tests := map[string]struct {
		stderr    string
		err       error
		wantSSH   bool
		wantRetry bool
		wantCat   string
	}{
		"banner timeout with exit 255": {
			stderr: "Connection timed out during banner exchange", err: exit255,
			wantSSH: true, wantRetry: true, wantCat: "banner-timeout",
		},
		"websocket eof": {
			stderr: "websocket: close 1006 (abnormal closure): unexpected EOF", err: exit255,
			wantSSH: true, wantRetry: true, wantCat: "websocket-eof",
		},
		"broken pipe": {
			stderr: "client_loop: send disconnect: Broken pipe", err: exit255,
			wantSSH: true, wantRetry: true, wantCat: "broken-pipe",
		},
		"network unreachable via stderr": {
			stderr: "Internal error occurred: dialing VM: dial tcp :22: connect: connection refused", err: exit255,
			wantSSH: true, wantRetry: true, wantCat: "network",
		},
		"context deadline": {
			stderr: "", err: context.DeadlineExceeded,
			wantSSH: true, wantRetry: true, wantCat: "timeout",
		},
		"auth failure is terminal": {
			stderr: "Permission denied (publickey)", err: exit255,
			wantSSH: true, wantRetry: false, wantCat: "authentication",
		},
		"connection reset by peer": {
			stderr: "read tcp 1.2.3.4:1234->5.6.7.8:6443: read: connection reset by peer\nexit status 255", err: exit255,
			wantSSH: true, wantRetry: true, wantCat: "connection-reset",
		},
		"connection closed by UNKNOWN": {
			stderr: "Connection closed by UNKNOWN port 65535", err: exit255,
			wantSSH: true, wantRetry: true, wantCat: "remote-host-closed",
		},
		"exit 255 with unknown stderr is not classified as transport": {
			stderr: "some totally unknown SSH failure message", err: exit255,
			wantSSH: false, wantRetry: false, wantCat: "",
		},
		"remote command exit 1 is not SSH": {
			stderr: "metadata already locked by 3814", err: exit1,
			wantSSH: false, wantRetry: false, wantCat: "",
		},
		"remote command exit 42 is not SSH": {
			stderr: "ROXAGENT_VERBOSE_ASSERT_FAILED", err: exit42,
			wantSSH: false, wantRetry: false, wantCat: "",
		},
		"plain error without ExitError is not SSH": {
			stderr: "something went wrong", err: errors.New("generic failure"),
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

func TestIsSudoPasswordPromptError(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		stderr string
		want   bool
	}{
		"terminal required": {
			stderr: "sudo: a terminal is required to read the password; either use the -S option to read from standard input or configure an askpass helper",
			want:   true,
		},
		"password required": {
			stderr: "sudo: a password is required",
			want:   true,
		},
		"other sudo failure": {
			stderr: "sudo: command not found",
			want:   false,
		},
		"unrelated error": {
			stderr: "Internal error occurred: dialing VM: dial tcp :22: connect: connection refused",
			want:   false,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, isSudoPasswordPromptError(tc.stderr))
		})
	}
}

func TestIsSSHAuthenticationFailure(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		stderr string
		want   bool
	}{
		"permission denied publickey": {
			stderr: "cloud-user@vmi.vm-rhel9.vm-scan-e2e-manual: Permission denied (publickey,gssapi-keyex,gssapi-with-mic).",
			want:   true,
		},
		"warning plus denied": {
			stderr: "Warning: Permanently added 'vmi.vm' (ED25519) to the list of known hosts.\ncloud-user@vmi.vm: Permission denied (publickey,gssapi-keyex,gssapi-with-mic).",
			want:   true,
		},
		"too many authentication failures": {
			stderr: "Received disconnect from UNKNOWN port 65535:2: Too many authentication failures\nDisconnected from UNKNOWN port 65535",
			want:   true,
		},
		"connection refused": {
			stderr: "Internal error occurred: dialing VM: dial tcp :22: connect: connection refused",
			want:   false,
		},
		"no route to host": {
			stderr: "Internal error occurred: dialing VM: dial tcp 10.131.0.48:22: connect: no route to host",
			want:   false,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, isSSHAuthenticationFailure(tc.stderr))
		})
	}
}

func TestIsSSHBannerTimeoutFailure(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		stderr string
		want   bool
	}{
		"banner exchange timeout": {
			stderr: "Connection timed out during banner exchange",
			want:   true,
		},
		"unknown port timeout": {
			stderr: "Connection to UNKNOWN port 65535 timed out",
			want:   true,
		},
		"permission denied": {
			stderr: "Permission denied (publickey,gssapi-keyex,gssapi-with-mic).",
			want:   false,
		},
		"connection refused": {
			stderr: "Internal error occurred: dialing VM: dial tcp :22: connect: connection refused",
			want:   false,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, isSSHBannerTimeoutFailure(tc.stderr))
		})
	}
}

func TestIsSSHNetworkUnreachableFailure(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		stderr string
		want   bool
	}{
		"no route to host": {
			stderr: "Internal error occurred: dialing VM: dial tcp 10.129.2.42:22: connect: no route to host",
			want:   true,
		},
		"connection refused": {
			stderr: "Internal error occurred: dialing VM: dial tcp :22: connect: connection refused",
			want:   true,
		},
		"permission denied": {
			stderr: "Permission denied (publickey,gssapi-keyex,gssapi-with-mic).",
			want:   false,
		},
		"banner timeout": {
			stderr: "Connection timed out during banner exchange",
			want:   false,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, isSSHNetworkUnreachableFailure(tc.stderr))
		})
	}
}

func TestIsSSHProbeTimeoutFailure(t *testing.T) {
	t.Parallel()

	require.True(t, isSSHProbeTimeoutFailure(context.DeadlineExceeded))
	require.True(t, isSSHProbeTimeoutFailure(fmt.Errorf("wrap: %w", context.DeadlineExceeded)))
	require.False(t, isSSHProbeTimeoutFailure(context.Canceled))
	require.False(t, isSSHProbeTimeoutFailure(errors.New("other")))
}

func TestSSHProbeFailureDetail(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		err    error
		stderr string
		want   string
	}{
		"stderr only": {
			stderr: "permission denied",
			want:   "permission denied",
		},
		"err only": {
			err:  context.DeadlineExceeded,
			want: context.DeadlineExceeded.Error(),
		},
		"stderr and err": {
			err:    context.DeadlineExceeded,
			stderr: "connection timed out",
			want:   "connection timed out (err: context deadline exceeded)",
		},
		"stderr already contains err": {
			err:    errors.New("exit status 255"),
			stderr: "permission denied; exit status 255",
			want:   "permission denied; exit status 255",
		},
		"neither": {
			want: "<no stderr>",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, sshProbeFailureDetail(tc.err, tc.stderr))
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

func TestSSHReachabilityPolicy_ClassifyFailureKeepsRetryingOnProbeTimeout(t *testing.T) {
	t.Parallel()

	policy := DefaultSSHReachabilityPolicy
	counters := &sshProbeCounters{}
	decision := policy.classifyFailure(counters, Virtctl{}, context.DeadlineExceeded, "")

	require.NoError(t, decision.terminalErr)
	require.Contains(t, decision.detail, "ssh probe deadline reached")
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
