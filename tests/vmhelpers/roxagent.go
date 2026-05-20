package vmhelpers

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Guest paths for staging and installing the roxagent binary during VM tests.
const (
	// DefaultRoxagentInstallPath is the path used when copying and running roxagent on the guest.
	DefaultRoxagentInstallPath = "/usr/local/bin/roxagent"
	roxagentStagingPath        = "/tmp/roxagent"
)

// ErrTerminalVSOCKUnavailable is returned when vsock is permanently unavailable on the guest (no retry).
var ErrTerminalVSOCKUnavailable = errors.New("terminal vsock unavailable")

// roxagentMaxAttempts is the number of times to retry roxagent before giving up.
const roxagentMaxAttempts = 3

// isVsockUnavailableOutput detects terminal vsock device errors in roxagent combined output.
func isVsockUnavailableOutput(output string) bool {
	lower := strings.ToLower(strings.TrimSpace(output))
	if !strings.Contains(lower, "vsock") {
		return false
	}
	return strings.Contains(lower, "no such device") ||
		strings.Contains(lower, "no such file or directory")
}

// verboseOutputHasOSFields reports whether a roxagent --verbose stdout
// contains at least one recognised OS-detection field.
func verboseOutputHasOSFields(stdout string) bool {
	return strings.Contains(stdout, `"detectedOs"`) ||
		strings.Contains(stdout, `"operatingSystem"`) ||
		strings.Contains(stdout, `"operating_system"`)
}

// CopyRoxagentBinary copies a local roxagent binary into the guest install path.
func CopyRoxagentBinary(ctx context.Context, virt Virtctl, namespace, vm, hostBinaryPath string) error {
	return retryOnSSHTransport(ctx, virt.Logf, "copy roxagent binary", func(ctx context.Context) error {
		stderr, err := virt.SCPTo(ctx, namespace, vm, hostBinaryPath, roxagentStagingPath)
		if err != nil {
			return fmt.Errorf("virtctl scp roxagent: %w: %s", err, strings.TrimSpace(stderr))
		}
		_, stderr, err = runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
			description:            "install roxagent binary",
			transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
		}, "sudo", "install", "-m", "0755", roxagentStagingPath, DefaultRoxagentInstallPath)
		if err != nil {
			return fmt.Errorf("install roxagent binary on guest: %w: %s", err, strings.TrimSpace(stderr))
		}
		_, _, _ = runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
			description:            "cleanup staged roxagent binary",
			transportRetryAttempts: 1,
		}, "rm", "-f", roxagentStagingPath)
		return nil
	})
}

// VerifyRoxagentInstalled runs `roxagent --help` on the guest to confirm the binary is
// present, executable, and resolvable in $PATH — all in a single SSH round-trip.
func VerifyRoxagentInstalled(ctx context.Context, virt Virtctl, namespace, vm string) error {
	return retryOnSSHTransport(ctx, virt.Logf, "verify roxagent installed", func(ctx context.Context) error {
		_, stderr, err := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
			description:            "verify roxagent installed",
			transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
		}, DefaultRoxagentInstallPath, "--help")
		if err != nil {
			return fmt.Errorf("roxagent --help: %w: %s", err, strings.TrimSpace(stderr))
		}
		return nil
	})
}

// RunRoxagentOnce runs roxagent on the guest with the given repo2cpe URL.
// It retries up to roxagentMaxAttempts times before giving up.
func RunRoxagentOnce(ctx context.Context, virt Virtctl, namespace, vm, repo2cpeURL string) error {
	envAssignment := fmt.Sprintf("ROXAGENT_REPO2CPE_URL=%s", repo2cpeURL)

	var lastErr error
	for attempt := range roxagentMaxAttempts {
		stdout, stderr, err := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
			description:            "run roxagent --verbose",
			transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
		}, "sudo", "env", envAssignment, DefaultRoxagentInstallPath, "--verbose")

		if err == nil {
			if !verboseOutputHasOSFields(stdout) {
				return fmt.Errorf("roxagent: verbose output OS assertion failed: no OS detection fields in output (stdout: %.200s)", strings.TrimSpace(stdout))
			}
			virt.Logf("roxagent completed (%d bytes stdout, %d bytes stderr)", len(stdout), len(stderr))
			return nil
		}
		combined := strings.TrimSpace(stdout + "\n" + stderr)
		if isVsockUnavailableOutput(combined) {
			return fmt.Errorf("%w: roxagent: no retry for vsock device error: %w (stderr: %s)",
				ErrTerminalVSOCKUnavailable, err, strings.TrimSpace(stderr))
		}
		lastErr = err
		virt.Logf("roxagent attempt %d/%d failed: %v", attempt+1, roxagentMaxAttempts, err)
	}
	return fmt.Errorf("roxagent: all %d attempts failed: %w", roxagentMaxAttempts, lastErr)
}
