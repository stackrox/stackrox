package vmhelpers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Guest paths for staging and installing the roxagent binary during VM tests.
const (
	// DefaultRoxagentInstallPath is the path used when copying and running roxagent on the guest.
	DefaultRoxagentInstallPath = "/usr/local/bin/roxagent"
	roxagentStagingPath        = "/tmp/roxagent"
	roxagentServeLogPath       = "/tmp/roxagent-serve.log"
	roxagentServePIDPath       = "/tmp/roxagent-serve.pid"
	roxagentServePort          = "818"
	roxagentListenReadyMarker  = "Listening on VSOCK port"
	roxagentListenPollInterval = 5 * time.Second
)

// ErrTerminalVSOCKUnavailable is returned when vsock is permanently unavailable on the guest (no retry).
var ErrTerminalVSOCKUnavailable = errors.New("terminal vsock unavailable")

// isVsockUnavailableOutput detects terminal vsock device errors in roxagent combined output.
func isVsockUnavailableOutput(output string) bool {
	lower := strings.ToLower(strings.TrimSpace(output))
	if !strings.Contains(lower, "vsock") {
		return false
	}
	return strings.Contains(lower, "no such device") ||
		strings.Contains(lower, "no such file or directory")
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

// EnsureRoxagentServing starts `roxagent serve` on the guest if it is not already
// running, then waits until the VSOCK listener is up. Sensor pulls reports from
// this long-running process; the test does not push index reports.
//
// repo2cpeURL is passed as --repo-cpe-url (serve does not read ROXAGENT_REPO2CPE_URL).
// The URL must not contain single quotes.
func EnsureRoxagentServing(ctx context.Context, virt Virtctl, namespace, vm, repo2cpeURL string) error {
	if strings.Contains(repo2cpeURL, "'") {
		return fmt.Errorf("repo2cpe URL must not contain single quotes: %q", repo2cpeURL)
	}
	if err := startRoxagentServeIfNeeded(ctx, virt, namespace, vm, repo2cpeURL); err != nil {
		return err
	}
	return waitRoxagentListening(ctx, virt, namespace, vm)
}

func startRoxagentServeIfNeeded(ctx context.Context, virt Virtctl, namespace, vm, repo2cpeURL string) error {
	return retryOnSSHTransport(ctx, virt.Logf, "start roxagent serve", func(ctx context.Context) error {
		// PID-file check avoids matching the sudo/bash wrapper that embeds the
		// same "roxagent serve" substring in its argv.
		script := fmt.Sprintf(`set -euo pipefail
if [ -f %[1]s ] && kill -0 "$(cat %[1]s)" 2>/dev/null; then
  echo ALREADY
  exit 0
fi
rm -f %[1]s %[2]s
nohup %[3]s serve --port %[4]s --host-path / --repo-cpe-url '%[5]s' >%[2]s 2>&1 </dev/null &
echo $! > %[1]s
sleep 1
if ! kill -0 "$(cat %[1]s)" 2>/dev/null; then
  echo 'roxagent serve exited immediately' >&2
  if [ -f %[2]s ]; then cat %[2]s >&2; fi
  exit 1
fi
echo STARTED
`, roxagentServePIDPath, roxagentServeLogPath, DefaultRoxagentInstallPath, roxagentServePort, repo2cpeURL)

		stdout, stderr, err := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
			description:            "start roxagent serve",
			transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
		}, "sudo", "bash", "-c", script)
		if err != nil {
			combined := strings.TrimSpace(stdout + "\n" + stderr)
			if isVsockUnavailableOutput(combined) {
				return fmt.Errorf("%w: roxagent serve: no retry for vsock device error: %w (stderr: %s)",
					ErrTerminalVSOCKUnavailable, err, strings.TrimSpace(stderr))
			}
			return fmt.Errorf("start roxagent serve: %w (stderr: %s)", err, strings.TrimSpace(stderr))
		}
		virt.Logf("roxagent serve: %s", strings.TrimSpace(stdout))
		return nil
	})
}

func waitRoxagentListening(ctx context.Context, virt Virtctl, namespace, vm string) error {
	script := fmt.Sprintf(`set -euo pipefail
if [ ! -f %[1]s ] || ! kill -0 "$(cat %[1]s)" 2>/dev/null; then
  echo DEAD
  if [ -f %[2]s ]; then cat %[2]s; fi
  exit 1
fi
if [ -f %[2]s ] && grep -q %[3]q %[2]s; then
  echo READY
  exit 0
fi
echo WAITING
exit 0
`, roxagentServePIDPath, roxagentServeLogPath, roxagentListenReadyMarker)

	for {
		stdout, stderr, err := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
			description:            "check roxagent serve readiness",
			transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
		}, "sudo", "bash", "-c", script)
		status := strings.TrimSpace(stdout)
		if err != nil {
			combined := strings.TrimSpace(stdout + "\n" + stderr)
			if isVsockUnavailableOutput(combined) {
				return fmt.Errorf("%w: roxagent serve died with vsock device error: %w (output: %s)",
					ErrTerminalVSOCKUnavailable, err, combined)
			}
			return fmt.Errorf("roxagent serve not running: %w (output: %s)", err, combined)
		}
		switch {
		case strings.HasPrefix(status, "READY"):
			virt.Logf("roxagent serve is listening on VSOCK")
			return nil
		case strings.HasPrefix(status, "WAITING"):
			virt.Logf("roxagent serve still starting (waiting for %q)", roxagentListenReadyMarker)
		default:
			return fmt.Errorf("unexpected roxagent serve readiness status %q (stderr: %s)", status, strings.TrimSpace(stderr))
		}

		timer := time.NewTimer(roxagentListenPollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("timed out waiting for roxagent serve to listen: %w", ctx.Err())
		case <-timer.C:
		}
	}
}
