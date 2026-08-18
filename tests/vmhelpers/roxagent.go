package vmhelpers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	// E2ERescanInterval is passed to serve --rescan-interval. It is the
	// minimum allowed interval and keeps the package-freshness e2e within a
	// reasonable wall-clock budget without reactive watching or restarts.
	E2ERescanInterval = 5 * time.Minute
	// E2EScraperPollInterval is the Sensor pull-mode scrape cadence used by
	// the VM e2e suite - floored at 1m.
	E2EScraperPollInterval = time.Minute
	// roxagentUnit is the Quadlet-generated systemd unit customers run.
	roxagentUnit               = "roxagent.service"
	roxagentListenPollInterval = 5 * time.Second
	guestPodmanAuthPath        = "/etc/containers/auth.json"
	guestPodmanAuthStagingPath = "/tmp/roxagent-podman-auth.json"
)

// ErrTerminalVSOCKUnavailable is returned when vsock is permanently unavailable on the guest (no retry).
var ErrTerminalVSOCKUnavailable = errors.New("terminal vsock unavailable")

// isVsockUnavailableOutput detects terminal vsock device errors in roxagent or
// Podman combined output (bare binary listen failures and Quadlet --device /dev/vsock).
func isVsockUnavailableOutput(output string) bool {
	lower := strings.ToLower(strings.TrimSpace(output))
	if !strings.Contains(lower, "vsock") {
		return false
	}
	return strings.Contains(lower, "no such device") ||
		strings.Contains(lower, "no such file or directory") ||
		(strings.Contains(lower, "/dev/vsock") && strings.Contains(lower, "not found"))
}

// EnsureRoxagentServing starts Quadlet `roxagent.service` if it is not already
// active, then waits until ActiveState is active (READY=1 after VSOCK listen).
func EnsureRoxagentServing(ctx context.Context, virt Virtctl, namespace, vm string) error {
	if err := startRoxagentServeIfNeeded(ctx, virt, namespace, vm); err != nil {
		return err
	}
	return waitRoxagentActive(ctx, virt, namespace, vm)
}

func startRoxagentServeIfNeeded(ctx context.Context, virt Virtctl, namespace, vm string) error {
	return retryOnSSHTransport(ctx, virt.Logf, "start roxagent serve", func(ctx context.Context) error {
		state, err := roxagentServeState(ctx, virt, namespace, vm)
		if err != nil {
			return err
		}
		if state == "active" || state == "activating" {
			virt.Logf("roxagent serve: %s already %s", roxagentUnit, state)
			return nil
		}

		if state == "failed" {
			_, _, _ = runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
				description:            "reset-failed roxagent.service",
				transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
			}, "sudo", "systemctl", "reset-failed", roxagentUnit)
		}

		_, stderr, err := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
			description:            "systemctl start roxagent.service",
			transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
		}, "sudo", "systemctl", "start", roxagentUnit)
		if err != nil {
			return wrapRoxagentUnitCmdError(ctx, virt, namespace, vm, "systemctl start "+roxagentUnit, err, stderr)
		}

		virt.Logf("roxagent serve: started %s", roxagentUnit)
		return nil
	})
}

// waitRoxagentActive polls until the Quadlet unit's ActiveState is active.
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

// RoxagentServeInvocationID returns the Quadlet unit's current systemd InvocationID.
// That UUID is unique per activation, so comparing it before and after a wait
// detects restarts.
func RoxagentServeInvocationID(ctx context.Context, virt Virtctl, namespace, vm string) (string, error) {
	id, err := systemctlShowProperty(ctx, virt, namespace, vm, roxagentUnit, "InvocationID")
	if err != nil {
		return "", err
	}
	if id == "" {
		return "", errors.New("roxagent.service InvocationID is empty")
	}
	return id, nil
}

// RoxagentServeDidNotRestart returns nil when the Quadlet unit still has beforeID
// as its InvocationID. A different ID means systemd started a new activation
// (stop/start, crash recovery, or an explicit restart).
func RoxagentServeDidNotRestart(ctx context.Context, virt Virtctl, namespace, vm, beforeID string) error {
	if beforeID == "" {
		return errors.New("before InvocationID is empty")
	}
	afterID, err := RoxagentServeInvocationID(ctx, virt, namespace, vm)
	if err != nil {
		return err
	}
	if afterID != beforeID {
		return fmt.Errorf("roxagent.service restarted during the wait: InvocationID changed from %s to %s",
			beforeID, afterID)
	}
	return nil
}

// roxagentServeState returns the Quadlet unit's ActiveState (active/inactive/failed/activating/…).
func roxagentServeState(ctx context.Context, virt Virtctl, namespace, vm string) (string, error) {
	state, err := systemctlShowProperty(ctx, virt, namespace, vm, roxagentUnit, "ActiveState")
	if err != nil {
		return "", err
	}
	if state == "" {
		return "unknown", nil
	}
	return state, nil
}

// wrapRoxagentUnitCmdError appends journal context and maps terminal vsock failures.
func wrapRoxagentUnitCmdError(ctx context.Context, virt Virtctl, namespace, vm, op string, err error, stderr string) error {
	logs := fetchRoxagentServeJournal(ctx, virt, namespace, vm)
	combined := strings.TrimSpace(stderr + "\n" + logs)
	if isVsockUnavailableOutput(combined) {
		return fmt.Errorf("%w: %s: %w (output: %s)", ErrTerminalVSOCKUnavailable, op, err, combined)
	}
	return fmt.Errorf("%s: %w (stderr: %s; journal: %s)", op, err, strings.TrimSpace(stderr), logs)
}

// fetchRoxagentServeJournal returns a short recent journal dump for error messages.
func fetchRoxagentServeJournal(ctx context.Context, virt Virtctl, namespace, vm string) string {
	stdout, stderr, err := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
		description:            "journalctl roxagent.service",
		transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
	}, "sudo", "journalctl", "-u", roxagentUnit, "-b", "--no-pager", "-o", "cat", "-n", "200")
	if err != nil {
		return strings.TrimSpace(stdout + "\n" + stderr)
	}
	return strings.TrimSpace(stdout)
}
