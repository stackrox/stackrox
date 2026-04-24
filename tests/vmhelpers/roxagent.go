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

// RoxagentRunConfig configures a single roxagent invocation (repo2cpe URL selection and binary path).
type RoxagentRunConfig struct {
	Repo2CPEPrimaryURL      string
	Repo2CPEFallbackURL     string
	Repo2CPEPrimaryAttempts int
	// RoxagentInstallPath may only be empty or DefaultRoxagentInstallPath. CopyRoxagentBinary and the
	// VerifyRoxagent* helpers always use DefaultRoxagentInstallPath; RunRoxagentOnce rejects any other
	// value so the run cannot target a path the suite never populated.
	RoxagentInstallPath string
	// Repo2CPEEnvVar is the environment variable name passed to roxagent (default ROXAGENT_REPO2CPE_URL).
	Repo2CPEEnvVar string
}

// RoxagentRunResult captures stdout/stderr and whether the fallback repo2cpe URL was used.
type RoxagentRunResult struct {
	Stdout       string
	Stderr       string
	UsedFallback bool
}

// roxagentPath returns RoxagentInstallPath when set, otherwise DefaultRoxagentInstallPath.
func roxagentPath(cfg RoxagentRunConfig) string {
	if cfg.RoxagentInstallPath != "" {
		return cfg.RoxagentInstallPath
	}
	return DefaultRoxagentInstallPath
}

// repo2cpeEnvName returns Repo2CPEEnvVar if set, otherwise ROXAGENT_REPO2CPE_URL.
func repo2cpeEnvName(cfg RoxagentRunConfig) string {
	if cfg.Repo2CPEEnvVar != "" {
		return cfg.Repo2CPEEnvVar
	}
	return "ROXAGENT_REPO2CPE_URL"
}

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

// VerifyRoxagentBinaryPresent checks that the roxagent file exists on the guest.
func VerifyRoxagentBinaryPresent(ctx context.Context, virt Virtctl, namespace, vm string) error {
	return retryOnSSHTransport(ctx, virt.Logf, "verify roxagent presence", func(ctx context.Context) error {
		_, stderr, err := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
			description:            "verify roxagent presence",
			transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
		}, "test", "-f", DefaultRoxagentInstallPath)
		if err != nil {
			return fmt.Errorf("roxagent presence: %w: %s", err, strings.TrimSpace(stderr))
		}
		return nil
	})
}

// VerifyRoxagentExecutable checks that the roxagent binary is executable.
func VerifyRoxagentExecutable(ctx context.Context, virt Virtctl, namespace, vm string) error {
	return retryOnSSHTransport(ctx, virt.Logf, "verify roxagent executable", func(ctx context.Context) error {
		_, stderr, err := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
			description:            "verify roxagent executable",
			transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
		}, "test", "-x", DefaultRoxagentInstallPath)
		if err != nil {
			return fmt.Errorf("roxagent executable: %w: %s", err, strings.TrimSpace(stderr))
		}
		return nil
	})
}

// VerifyRoxagentInstallPath checks that `command -v roxagent` resolves to the canonical install path.
func VerifyRoxagentInstallPath(ctx context.Context, virt Virtctl, namespace, vm string) error {
	return retryOnSSHTransport(ctx, virt.Logf, "verify roxagent install path", func(ctx context.Context) error {
		stdout, stderr, err := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
			description:            "verify roxagent install path",
			transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
		}, "command", "-v", "roxagent")
		if err != nil {
			return fmt.Errorf("roxagent install path: %w: %s", err, strings.TrimSpace(stderr))
		}
		if got := strings.TrimSpace(stdout); got != DefaultRoxagentInstallPath {
			return fmt.Errorf("roxagent install path: got %q, want %q", got, DefaultRoxagentInstallPath)
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
	if cfg.RoxagentInstallPath != "" && cfg.RoxagentInstallPath != DefaultRoxagentInstallPath {
		return nil, fmt.Errorf(
			"vmhelpers RunRoxagentOnce: RoxagentInstallPath %q is not supported (CopyRoxagentBinary and VerifyRoxagent* only use %q); omit RoxagentInstallPath or set it to DefaultRoxagentInstallPath",
			cfg.RoxagentInstallPath, DefaultRoxagentInstallPath,
		)
	}
	maxPrimary := cfg.Repo2CPEPrimaryAttempts
	if maxPrimary < 0 {
		maxPrimary = 0
	}
	bin := roxagentPath(cfg)
	envName := repo2cpeEnvName(cfg)
	if err := ensureVsockReady(ctx, virt, namespace, vm, "roxagent run"); err != nil {
		return nil, err
	}

	var lastStderr string

	for attempt := 0; ; attempt++ {
		url := chooseRepo2CPESource(attempt, maxPrimary, cfg.Repo2CPEPrimaryURL, cfg.Repo2CPEFallbackURL)
		usedFallback := url == cfg.Repo2CPEFallbackURL && cfg.Repo2CPEFallbackURL != ""
		envAssignment := fmt.Sprintf("%s=%s", envName, url)

		stdout, stderr, err := runSSHCommandWithFramework(ctx, virt, namespace, vm, sshCommandRunOptions{
			description:            "run roxagent --verbose",
			transportRetryAttempts: rhsmPrecheckSSHRetryThreshold,
		}, "sudo", "env", envAssignment, bin, "--verbose")
		lastStderr = stderr

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
			return nil, fmt.Errorf("roxagent: %w (stderr: %s)", err, strings.TrimSpace(lastStderr))
		}
	}
}
