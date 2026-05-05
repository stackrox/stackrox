package vmhelpers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

// SSH reachability and guest wait tuning: probe cadence, per-category stall thresholds,
// single-probe timeout, and wait-loop logging/truncation for cloud-init and similar polls.
const (
	// sshReachablePollInterval is the sleep between consecutive SSH probe attempts.
	sshReachablePollInterval = 10 * time.Second
	// sshAuthFailureThreshold is the number of consecutive "permission denied" probe
	// failures required to classify credentials as stale/missing before recovery kicks in.
	sshAuthFailureThreshold = 3
	// sshBannerTimeoutThreshold is the number of consecutive "banner exchange timeout"
	// probe failures required to classify SSH connectivity as stalled.
	sshBannerTimeoutThreshold = 6
	// sshNetworkUnreachableThreshold is the number of consecutive network failures
	// ("no route to host"/"connection refused") required to classify connectivity as stalled.
	sshNetworkUnreachableThreshold = 36
	// sshProbeTimeoutThreshold is the number of consecutive per-probe timeout failures
	// required to classify SSH connectivity as stalled.
	sshProbeTimeoutThreshold = 6
	// sshProbeAttemptTimeout bounds one SSH probe attempt so wait-loop diagnostics
	// continue even when a single virtctl invocation gets stuck.
	sshProbeAttemptTimeout = 20 * time.Second
)

// passwordlessSudoRequirementHint is appended to errors when cloud-init/sudo checks need NOPASSWD sudo for the SSH user.
const passwordlessSudoRequirementHint = `configure guest cloud-init with sudo: "ALL=(ALL) NOPASSWD:ALL"`

// ErrSSHAuthenticationFailed indicates repeated SSH permission-denied failures.
// Callers may recreate/reconfigure a VM and retry once this error is returned.
var ErrSSHAuthenticationFailed = errors.New("ssh authentication failed")

// ErrSSHConnectivityStalled indicates repeated SSH banner timeout failures.
// Callers may recreate/reconfigure a VM and retry once this error is returned.
var ErrSSHConnectivityStalled = errors.New("ssh connectivity stalled")

// isSudoPasswordPromptError reports stderr patterns indicating sudo needs a TTY or password (non-passwordless sudo).
func isSudoPasswordPromptError(stderr string) bool {
	stderr = strings.ToLower(strings.TrimSpace(stderr))
	return strings.Contains(stderr, "sudo: a terminal is required to read the password") ||
		strings.Contains(stderr, "sudo: a password is required")
}

// isSSHAuthenticationFailure detects permission denied / too many auth failures in SSH client stderr.
func isSSHAuthenticationFailure(stderr string) bool {
	stderr = strings.ToLower(strings.TrimSpace(stderr))
	return strings.Contains(stderr, "permission denied (publickey") ||
		strings.Contains(stderr, "too many authentication failures")
}

// isSSHBannerTimeoutFailure detects SSH banner exchange timeouts typical of a slow or wedged sshd path.
func isSSHBannerTimeoutFailure(stderr string) bool {
	stderr = strings.ToLower(strings.TrimSpace(stderr))
	return strings.Contains(stderr, "connection timed out during banner exchange") ||
		strings.Contains(stderr, "connection to unknown port 65535 timed out")
}

// isSSHNetworkUnreachableFailure detects immediate connect failures (no route / connection refused) from the SSH client.
func isSSHNetworkUnreachableFailure(stderr string) bool {
	stderr = strings.ToLower(strings.TrimSpace(stderr))
	return strings.Contains(stderr, "connect: no route to host") ||
		strings.Contains(stderr, "connect: connection refused")
}

// sshUserForDiagnostic returns the configured virtctl SSH username or a placeholder for log and error text.
func sshUserForDiagnostic(virt Virtctl) string {
	if user := strings.TrimSpace(virt.Username); user != "" {
		return user
	}
	return "<default ssh user>"
}

// isSSHProbeTimeoutFailure is true when the probe failed because its context deadline was exceeded.
func isSSHProbeTimeoutFailure(err error) bool {
	return errors.Is(err, context.DeadlineExceeded)
}

// sshProbeFailureDetail combines stderr and error into a single diagnostic string for logging and stall decisions.
func sshProbeFailureDetail(err error, stderr string) string {
	stderr = strings.TrimSpace(stderr)
	switch {
	case stderr == "" && err == nil:
		return "<no stderr>"
	case stderr == "":
		return err.Error()
	case err == nil:
		return stderr
	case strings.Contains(stderr, err.Error()):
		return stderr
	default:
		return fmt.Sprintf("%s (err: %v)", stderr, err)
	}
}

// WaitForSSHReachable polls SSH until `true` succeeds or policy classifies a terminal auth/connectivity failure.
func WaitForSSHReachable(t testing.TB, ctx context.Context, virt Virtctl, namespace, vm string) error {
	t.Helper()
	policy := defaultSSHReachabilityPolicy
	attempts := 0
	counters := &sshProbeCounters{}
	lastDetail := ""
	desc := fmt.Sprintf("wait SSH %s/%s reachable", namespace, vm)
	maxAttempts, maxKnown := estimateMaxPollAttempts(ctx, policy.pollInterval)
	err := wait.PollUntilContextCancel(ctx, policy.pollInterval, true, func(ctx context.Context) (bool, error) {
		attempts++
		stderr, err := runSSHReachabilityProbe(ctx, policy, virt, namespace, vm)
		if err == nil {
			counters.resetAll()
			lastDetail = "ssh command succeeded"
			logWaitAttempt(t, desc, attempts, maxAttempts, maxKnown, lastDetail)
			return true, nil
		}
		decision := policy.classifyFailure(counters, virt, err, stderr)
		lastDetail = decision.detail
		logWaitAttempt(t, desc, attempts, maxAttempts, maxKnown, lastDetail)
		if decision.terminalErr != nil {
			return false, decision.terminalErr
		}
		return false, nil
	})
	if err == nil {
		return nil
	}
	if lastDetail != "" {
		return fmt.Errorf("wait SSH reachable for %s/%s failed after %d poll(s): %w (last detail: %s)", namespace, vm, attempts, err, truncateWaitDetail(lastDetail))
	}
	return fmt.Errorf("wait SSH reachable for %s/%s failed after %d poll(s): %w", namespace, vm, attempts, err)
}

// WaitForCloudInitFinished runs `sudo cloud-init status --wait` on the guest with SSH transport retries.
func WaitForCloudInitFinished(ctx context.Context, virt Virtctl, namespace, vm string) error {
	return retryOnSSHTransport(ctx, virt.Logf, "cloud-init status --wait", func(ctx context.Context) error {
		_, stderr, err := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
			description:            "cloud-init status --wait",
			transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
		}, "sudo", "cloud-init", "status", "--wait")
		if err != nil {
			stderr = strings.TrimSpace(stderr)
			if isSudoPasswordPromptError(stderr) {
				return fmt.Errorf("cloud-init status --wait on %s/%s requires passwordless sudo for ssh user %q (%s): %w: %s",
					namespace, vm, sshUserForDiagnostic(virt), passwordlessSudoRequirementHint, err, stderr)
			}
			return fmt.Errorf("cloud-init status --wait: %w: %s", err, stderr)
		}
		return nil
	})
}

// VerifySudoWorks checks passwordless sudo via `sudo -n true` with SSH transport retries.
func VerifySudoWorks(ctx context.Context, virt Virtctl, namespace, vm string) error {
	return retryOnSSHTransport(ctx, virt.Logf, "passwordless sudo check", func(ctx context.Context) error {
		_, stderr, err := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
			description:            "passwordless sudo check",
			transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
		}, "sudo", "-n", "true")
		if err != nil {
			stderr = strings.TrimSpace(stderr)
			if isSudoPasswordPromptError(stderr) {
				return fmt.Errorf("passwordless sudo check failed for %s/%s ssh user %q (%s): %w: %s",
					namespace, vm, sshUserForDiagnostic(virt), passwordlessSudoRequirementHint, err, stderr)
			}
			return fmt.Errorf("sudo -n true: %w: %s", err, stderr)
		}
		return nil
	})
}
