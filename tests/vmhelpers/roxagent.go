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
	roxagentServePort          = "818"
	// Transient systemd unit started by EnsureRoxagentServing (systemd-run --unit=...).
	roxagentE2EUnit            = "roxagent-e2e.service"
	roxagentE2EUnitName        = "roxagent-e2e"
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

// EnsureRoxagentServing starts `roxagent serve` on the guest via a transient systemd
// unit if it is not already active, then waits until the unit is active.
// With Type=notify, active means roxagent has bound the VSOCK listener and sent
// READY=1 to systemd. Sensor then pulls reports from this long-running process.
//
// repo2cpeURL is passed as --repo-cpe-url (serve does not read ROXAGENT_REPO2CPE_URL).
func EnsureRoxagentServing(ctx context.Context, virt Virtctl, namespace, vm, repo2cpeURL string) error {
	if err := startRoxagentServeIfNeeded(ctx, virt, namespace, vm, repo2cpeURL); err != nil {
		return err
	}
	return waitRoxagentActive(ctx, virt, namespace, vm)
}

func startRoxagentServeIfNeeded(ctx context.Context, virt Virtctl, namespace, vm, repo2cpeURL string) error {
	return retryOnSSHTransport(ctx, virt.Logf, "start roxagent serve", func(ctx context.Context) error {
		state, err := roxagentServeState(ctx, virt, namespace, vm)
		if err != nil {
			return err
		}
		if state == "active" || state == "activating" {
			virt.Logf("roxagent serve: %s already %s", roxagentE2EUnit, state)
			return nil
		}

		// systemd-run refuses to reuse a unit name that still exists (failed or
		// stopped). Clear leftovers from a prior attempt, then start fresh.
		_, _, _ = runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
			description:            "stop leftover roxagent-e2e unit",
			transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
		}, "sudo", "systemctl", "stop", roxagentE2EUnit)
		_, _, _ = runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
			description:            "reset-failed roxagent-e2e unit",
			transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
		}, "sudo", "systemctl", "reset-failed", roxagentE2EUnit)

		_, stderr, err := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
			description:            "systemd-run roxagent serve",
			transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
		}, "sudo", "systemd-run",
			"--unit="+roxagentE2EUnitName,
			"--property=Type=notify",
			DefaultRoxagentInstallPath, "serve",
			"--port", roxagentServePort,
			"--host-path", "/",
			"--repo-cpe-url", repo2cpeURL,
		)
		if err != nil {
			logs := fetchRoxagentServeJournal(ctx, virt, namespace, vm)
			combined := strings.TrimSpace(stderr + "\n" + logs)
			if isVsockUnavailableOutput(combined) {
				return fmt.Errorf("%w: systemd-run roxagent serve: %w (output: %s)",
					ErrTerminalVSOCKUnavailable, err, combined)
			}
			return fmt.Errorf("systemd-run roxagent serve: %w (stderr: %s; journal: %s)",
				err, strings.TrimSpace(stderr), logs)
		}

		virt.Logf("roxagent serve: systemd-run accepted %s", roxagentE2EUnit)
		return nil
	})
}

// waitRoxagentActive polls until the e2e unit's ActiveState is active.
// Type=notify keeps the unit in activating until READY=1 after VSOCK listen.
func waitRoxagentActive(ctx context.Context, virt Virtctl, namespace, vm string) error {
	for {
		state, err := roxagentServeState(ctx, virt, namespace, vm)
		if err != nil {
			return err
		}
		switch state {
		case "active":
			virt.Logf("roxagent serve is active (VSOCK listener ready)")
			return nil
		case "activating":
			virt.Logf("roxagent serve still starting (unit=%s, waiting for Type=notify READY)", state)
		default:
			logs := fetchRoxagentServeJournal(ctx, virt, namespace, vm)
			if isVsockUnavailableOutput(logs) {
				return fmt.Errorf("%w: roxagent serve unit %q (journal: %s)",
					ErrTerminalVSOCKUnavailable, state, logs)
			}
			return fmt.Errorf("roxagent serve unit %q (journal: %s)", state, logs)
		}

		timer := time.NewTimer(roxagentListenPollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("timed out waiting for roxagent serve to become active: %w", ctx.Err())
		case <-timer.C:
		}
	}
}

// systemctlShowProperty runs `systemctl show -p <property> --value <unit>` on the guest and
// returns the property's trimmed value. Unlike `is-active`/`is-enabled`, `show` always exits 0
// for any unit systemd knows about — loaded or not, active or not — so a non-zero exit here
// reliably means a genuine remote-command problem, not an expected state value. That makes it
// the preferred way to read any systemd unit property from the guest; add new lookups (like
// roxagentServeState below) on top of it rather than parsing the output or exit code of
// state-specific subcommands such as `is-active`.
func systemctlShowProperty(ctx context.Context, virt Virtctl, namespace, vm, unit, property string) (string, error) {
	stdout, stderr, err := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
		description:            fmt.Sprintf("systemctl show -p %s %s", property, unit),
		transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
	}, "sudo", "systemctl", "show", "-p", property, "--value", unit)
	if err != nil {
		return "", fmt.Errorf("systemctl show -p %s %s: %w (stdout: %s; stderr: %s)",
			property, unit, err, strings.TrimSpace(stdout), strings.TrimSpace(stderr))
	}
	return strings.TrimSpace(stdout), nil
}

// roxagentServeState returns the e2e unit's ActiveState (active/inactive/failed/activating/…).
func roxagentServeState(ctx context.Context, virt Virtctl, namespace, vm string) (string, error) {
	state, err := systemctlShowProperty(ctx, virt, namespace, vm, roxagentE2EUnit, "ActiveState")
	if err != nil {
		return "", err
	}
	if state == "" {
		return "unknown", nil
	}
	return state, nil
}

// fetchRoxagentServeJournal returns a short recent journal dump for error messages.
func fetchRoxagentServeJournal(ctx context.Context, virt Virtctl, namespace, vm string) string {
	stdout, stderr, err := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
		description:            "journalctl roxagent-e2e",
		transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
	}, "sudo", "journalctl", "-u", roxagentE2EUnit, "-b", "--no-pager", "-o", "cat", "-n", "200")
	if err != nil {
		return strings.TrimSpace(stdout + "\n" + stderr)
	}
	return strings.TrimSpace(stdout)
}
