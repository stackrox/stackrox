package vmhelpers

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Default retry count and backoff between SSH transport retries inside runSSHCommandWithFramework.
const (
	defaultSSHTransportRetryAttempts = 3
	defaultSSHTransportRetryInterval = 2 * time.Second
)

// errSSHTransport is a sentinel that wraps errors originating from the SSH
// transport layer (banner timeout, network unreachable, authentication failure)
// as opposed to errors returned by the remote command itself.
// Callers can use errors.Is(err, errSSHTransport) to distinguish transport
// failures from remote-command failures and apply different retry strategies.
var errSSHTransport = errors.New("SSH transport error")

// sshCommandRunOptions configures logging, transport retries, and backoff for runSSHCommandWithFramework.
type sshCommandRunOptions struct {
	description            string
	transportRetryAttempts int
	retryInterval          time.Duration
	// suppressLog disables command-level logging for this call (use when the
	// command contains secrets such as activation keys).
	suppressLog bool
}

// isExitCode255 reports whether err wraps an *exec.ExitError with exit code 255.
// OpenSSH (and virtctl, which wraps it) uses exit code 255 exclusively for
// SSH transport failures; remote commands return 0–254.
func isExitCode255(err error) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr) && exitErr.ExitCode() == 255
}

// classifySSHStderrCategory returns a descriptive category for known SSH
// transport stderr patterns so that logging stays specific. If no known
// pattern matches, it returns "".
func classifySSHStderrCategory(stderr string) (category string, retryable bool) {
	if isSSHAuthenticationFailure(stderr) {
		return "authentication", false
	}
	if isSSHBannerTimeoutFailure(stderr) {
		return "banner-timeout", true
	}
	if isSSHNetworkUnreachableFailure(stderr) {
		return "network", true
	}
	lower := strings.ToLower(strings.TrimSpace(stderr))
	switch {
	case strings.Contains(lower, "websocket: close 1006"):
		return "websocket-eof", true
	case strings.Contains(lower, "unexpected eof"):
		return "unexpected-eof", true
	case strings.Contains(lower, "broken pipe"):
		return "broken-pipe", true
	case strings.Contains(lower, "closed by remote host"),
		strings.Contains(lower, "connection closed by"):
		return "remote-host-closed", true
	case strings.Contains(lower, "internal error occurred: dialing vm"):
		return "dialing-vm", true
	case strings.Contains(lower, "connection reset by peer"):
		return "connection-reset", true
	default:
		return "", true
	}
}

// classifySSHFailure decides whether a failure is SSH transport-level (vs remote command) and if retrying helps.
func classifySSHFailure(stderr string, err error) (isSSH bool, retryable bool, category string) {
	if err == nil {
		return false, false, ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true, true, "timeout"
	}

	// Check stderr for a known pattern first — gives us the most specific
	// category for logging regardless of exit code.
	cat, catRetryable := classifySSHStderrCategory(stderr)

	// Two independent signals: either a known stderr pattern OR exit code
	// 255 is sufficient to classify this as an SSH transport error.
	if cat != "" {
		return true, catRetryable, cat
	}
	if isExitCode255(err) {
		return true, true, "transport-exit-255"
	}

	return false, false, ""
}

// sshTransportRetryInterval is the pause between retries in retryOnSSHTransport after transport errors.
const sshTransportRetryInterval = 10 * time.Second

// retryOnSSHTransport retries fn whenever it returns an errSSHTransport error.
// Non-transport errors and nil are returned immediately. The retry loop is
// bounded by ctx — callers should set an appropriate deadline/timeout.
func retryOnSSHTransport(ctx context.Context, logf func(string, ...any), desc string, fn func(ctx context.Context) error) error {
	var lastErr error
	for attempt := 1; ; attempt++ {
		lastErr = fn(ctx)
		if lastErr == nil || !errors.Is(lastErr, errSSHTransport) {
			return lastErr
		}
		if logf != nil {
			logf("%s: SSH transport error (attempt %d), retrying in %s: %v",
				desc, attempt, sshTransportRetryInterval, lastErr)
		}
		timer := time.NewTimer(sshTransportRetryInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("%s: context expired while retrying SSH transport error: %w (last: %v)",
				desc, ctx.Err(), lastErr)
		case <-timer.C:
		}
	}
}

// runSSHCommandWithFramework runs virt.SSH with transport classification, bounded retries, and optional logging.
func runSSHCommandWithFramework(ctx context.Context, virt Virtctl, namespace, vm string, opts sshCommandRunOptions, command ...string) (stdout, stderr string, err error) {
	if opts.suppressLog {
		virt.Logf = nil
	}
	attempts := opts.transportRetryAttempts
	if attempts <= 0 {
		attempts = defaultSSHTransportRetryAttempts
	}
	interval := opts.retryInterval
	if interval <= 0 {
		interval = defaultSSHTransportRetryInterval
	}
	description := strings.TrimSpace(opts.description)
	if description == "" {
		description = "ssh command"
	}

	for attempt := 1; attempt <= attempts; attempt++ {
		stdout, stderr, err = virt.SSH(ctx, namespace, vm, command...)
		if err == nil {
			return stdout, stderr, nil
		}

		isSSH, retryable, category := classifySSHFailure(stderr, err)
		if !isSSH {
			return stdout, stderr, err
		}
		if !retryable {
			return stdout, stderr, fmt.Errorf("%w: %s on %s/%s: terminal SSH %s failure: %w",
				errSSHTransport, description, namespace, vm, category, err)
		}
		if attempt >= attempts {
			return stdout, stderr, fmt.Errorf("%w: %s on %s/%s: retryable SSH %s failure persisted after %d attempt(s): %w",
				errSSHTransport, description, namespace, vm, category, attempts, err)
		}

		if virt.Logf != nil {
			virt.Logf("%s on %s/%s: retryable SSH %s failure (attempt %d/%d): %s",
				description, namespace, vm, category, attempt, attempts, formatGuestCommandOutputForError(stderr))
		}

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return stdout, stderr, fmt.Errorf("%w: %s on %s/%s: context done during SSH retry backoff: %w",
				errSSHTransport, description, namespace, vm, ctx.Err())
		case <-timer.C:
		}
	}

	return stdout, stderr, err
}
