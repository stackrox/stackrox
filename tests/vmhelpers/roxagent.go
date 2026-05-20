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

// RoxagentRunConfig configures a single roxagent invocation (repo2cpe URL selection).
type RoxagentRunConfig struct {
	Repo2CPEPrimaryURL      string
	Repo2CPEFallbackURL     string
	Repo2CPEPrimaryAttempts int
}

// RoxagentRunResult captures stdout/stderr and whether the fallback repo2cpe URL was used.
type RoxagentRunResult struct {
	Stdout       string
	Stderr       string
	UsedFallback bool
}

// roxagentRepo2CPEEnvVar is the environment variable name passed to roxagent on the guest.
const roxagentRepo2CPEEnvVar = "ROXAGENT_REPO2CPE_URL"

// chooseRepo2CPESource returns primaryURL until attemptsMade reaches maxPrimaryAttempts, then fallbackURL.
func chooseRepo2CPESource(attemptsMade, maxPrimaryAttempts int, primaryURL, fallbackURL string) string {
	if attemptsMade < maxPrimaryAttempts {
		return primaryURL
	}
	return fallbackURL
}

// isVsockUnavailableOutput detects terminal vsock device errors in roxagent combined output.
func isVsockUnavailableOutput(output string) bool {
	lower := strings.ToLower(strings.TrimSpace(output))
	if !strings.Contains(lower, "vsock") {
		return false
	}
	return strings.Contains(lower, "no such device") ||
		strings.Contains(lower, "no such file or directory")
}

// IsTerminalVSOCKUnavailableError reports whether err wraps ErrTerminalVSOCKUnavailable.
func IsTerminalVSOCKUnavailableError(err error) bool {
	return errors.Is(err, ErrTerminalVSOCKUnavailable)
}

// verboseOutputHasOSFields reports whether a roxagent --verbose stdout
// contains at least one recognised OS-detection field.
func verboseOutputHasOSFields(stdout string) bool {
	return strings.Contains(stdout, `"detectedOs"`) ||
		strings.Contains(stdout, `"operatingSystem"`) ||
		strings.Contains(stdout, `"operating_system"`)
}

// buildRoxagentInstallArgs is the remote argv for `sudo install` to place the staged binary at dst.
func buildRoxagentInstallArgs(src, dst string) []string {
	return []string{"sudo", "install", "-m", "0755", src, dst}
}

// VerboseOutputLooksLikeReport returns true when stdout appears to contain a known
// structured roxagent report shape (legacy scan-shaped or indexReport-shaped JSON).
func VerboseOutputLooksLikeReport(stdout string) bool {
	s := strings.TrimSpace(stdout)
	if s == "" {
		return false
	}
	if strings.Contains(s, `"indexReport"`) || strings.Contains(s, `"discoveredData"`) {
		return true
	}
	return strings.Contains(s, `"components"`) &&
		(strings.Contains(s, `"scan"`) || strings.Contains(s, `"operatingSystem"`) || strings.Contains(s, `"operating_system"`))
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
		}, buildRoxagentInstallArgs(roxagentStagingPath, DefaultRoxagentInstallPath)...)
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

// RunRoxagentOnce runs roxagent on the guest. It uses Repo2CPEPrimaryURL for each attempt
// index in [0, Repo2CPEPrimaryAttempts) (i.e. up to Repo2CPEPrimaryAttempts primary runs when that
// count is positive), stopping as soon as one run succeeds. If all primary runs fail, it performs
// exactly one further run using Repo2CPEFallbackURL. When Repo2CPEPrimaryAttempts is zero, the
// first run uses the fallback URL only.
func RunRoxagentOnce(ctx context.Context, virt Virtctl, namespace, vm string, cfg RoxagentRunConfig) (*RoxagentRunResult, error) {
	maxPrimary := cfg.Repo2CPEPrimaryAttempts
	if maxPrimary < 0 {
		maxPrimary = 0
	}
	if err := ensureVsockReady(ctx, virt, namespace, vm, "roxagent run"); err != nil {
		return nil, err
	}

	for attempt := 0; ; attempt++ {
		url := chooseRepo2CPESource(attempt, maxPrimary, cfg.Repo2CPEPrimaryURL, cfg.Repo2CPEFallbackURL)
		usedFallback := url == cfg.Repo2CPEFallbackURL && cfg.Repo2CPEFallbackURL != ""
		envAssignment := fmt.Sprintf("%s=%s", roxagentRepo2CPEEnvVar, url)

		stdout, stderr, err := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
			description:            "run roxagent --verbose",
			transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
		}, "sudo", "env", envAssignment, DefaultRoxagentInstallPath, "--verbose")

		if err == nil {
			if !verboseOutputHasOSFields(stdout) {
				return nil, fmt.Errorf("roxagent: verbose output OS assertion failed: no OS detection fields in output (stdout: %.200s)", strings.TrimSpace(stdout))
			}
			return &RoxagentRunResult{
				Stdout:       stdout,
				Stderr:       stderr,
				UsedFallback: usedFallback,
			}, nil
		}
		combined := strings.TrimSpace(stdout + "\n" + stderr)
		if isVsockUnavailableOutput(combined) {
			return nil, fmt.Errorf("%w: roxagent: no retry for vsock device error: %w (stderr: %s)",
				ErrTerminalVSOCKUnavailable, err, strings.TrimSpace(stderr))
		}

		// attempt indices 0..maxPrimary-1 use primary; attempt maxPrimary uses fallback. Stop after fallback fails.
		if attempt >= maxPrimary {
			return nil, fmt.Errorf("roxagent: %w (stderr: %s)", err, strings.TrimSpace(stderr))
		}
	}
}
