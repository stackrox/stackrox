package vmhelpers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

// defaultLocalSSHOpts are SSH client options passed on every virtctl ssh/scp invocation via --local-ssh-opts.
var defaultLocalSSHOpts = []string{
	"-o StrictHostKeyChecking=no",
	"-o IdentitiesOnly=yes",
	"-o ConnectTimeout=5",
}

// defaultVirtctlHeartbeatInterval is the interval between progress log lines while waiting for a virtctl command.
const defaultVirtctlHeartbeatInterval = 30 * time.Second

const inlineLogMaxHeadTailLines = 100

// Virtctl runs virtctl subcommands with optional per-call timeout.
type Virtctl struct {
	Path           string
	IdentityFile   string
	Username       string
	CommandTimeout time.Duration
	// KnownHostsFile, when set, points SSH at a real known_hosts file so that
	// host keys learned on the first connection suppress the "Permanently added"
	// warning on subsequent ones. Leave empty to use /dev/null (every connection
	// warns). Use CreateKnownHostsFile to create a per-test temp file.
	KnownHostsFile string
	// Logf is optional. When provided, each remote command logs start/heartbeat/completion.
	Logf func(format string, args ...any)
	// HeartbeatInterval controls "still running" log cadence for long commands.
	// Zero uses defaultVirtctlHeartbeatInterval.
	HeartbeatInterval time.Duration
	// LogSuccessfulStreams, when true and Logf is set, includes truncated stdout/stderr
	// on successful remote commands (same formatting as failures). Default is false so
	// CI logs only record byte sizes unless a run fails or this is enabled for debugging.
	LogSuccessfulStreams bool
}

func (v Virtctl) knownHostsFile() string {
	if v.KnownHostsFile != "" {
		return v.KnownHostsFile
	}
	return "/dev/null"
}

// CreateKnownHostsFile creates an empty temp file suitable for KnownHostsFile
// and registers its removal via t.Cleanup. The first SSH connection populates
// it with the VM's host key; subsequent connections find the key and skip the
// "Permanently added" warning.
func CreateKnownHostsFile(t testing.TB) string {
	t.Helper()
	f, err := os.CreateTemp("", "virtctl-known-hosts-*")
	if err != nil {
		t.Logf("WARNING: failed to create known_hosts temp file, falling back to /dev/null: %v", err)
		return "/dev/null"
	}
	name := f.Name()
	_ = f.Close()
	t.Cleanup(func() { _ = os.Remove(name) })
	return name
}

// run starts a virtctl (or argv[0]) subprocess, captures stdout/stderr, and honors optional logging and heartbeats.
func (v Virtctl) run(ctx context.Context, args []string) (stdout string, stderr string, err error) {
	if v.CommandTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, v.CommandTimeout)
		defer cancel()
	}
	if len(args) == 0 {
		return "", "", errors.New("virtctl: empty args")
	}
	cmd := exec.Command(args[0], args[1:]...)
	configureVirtctlCmdForCancellation(cmd)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	start := time.Now()
	summary := summarizeVirtctlCommand(args)
	var (
		stopHeartbeat chan struct{}
		hbWG          sync.WaitGroup
	)
	if v.Logf != nil {
		v.Logf("remote command start: %s (deadline in %s)", summary, formatDeadlineRemaining(ctx))
		stopHeartbeat = make(chan struct{})
		interval := v.HeartbeatInterval
		if interval <= 0 {
			interval = defaultVirtctlHeartbeatInterval
		}
		hbWG.Go(func() {
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				select {
				case <-stopHeartbeat:
					return
				case <-ticker.C:
					if vmTarget, ok := virtctlSSHVMTarget(args); ok {
						v.Logf("remote command still running (%s elapsed) on VM %s", time.Since(start).Round(time.Second), vmTarget)
					} else {
						v.Logf("remote command still running (%s elapsed): %s", time.Since(start).Round(time.Second), summary)
					}
				}
			}
		})
	}
	if err = cmd.Start(); err != nil {
		if stopHeartbeat != nil {
			close(stopHeartbeat)
			hbWG.Wait()
		}
		stdoutStr := outBuf.String()
		stderrStr := errBuf.String()
		if v.Logf != nil {
			v.Logf("remote command could not start: %s (result=%v)", summary, err)
		}
		return stdoutStr, stderrStr, err
	}
	err = waitForCommandWithContext(ctx, cmd)

	if stopHeartbeat != nil {
		close(stopHeartbeat)
		hbWG.Wait()
	}
	stdoutStr := outBuf.String()
	stderrStr := errBuf.String()
	if v.Logf != nil {
		elapsed := time.Since(start).Round(time.Second)
		if err != nil {
			v.Logf("remote command not successful in %s: %s (result=%v stdout=%dB stderr=%dB)\n%s",
				elapsed, summary, err, len(stdoutStr), len(stderrStr), formatRemoteCommandStreamsForInlineLog(stdoutStr, stderrStr))
		} else {
			if v.LogSuccessfulStreams {
				v.Logf("remote command complete in %s: %s (stdout=%dB stderr=%dB)\n%s",
					elapsed, summary, len(stdoutStr), len(stderrStr), formatRemoteCommandStreamsForInlineLog(stdoutStr, stderrStr))
			} else {
				v.Logf("remote command complete in %s: %s (stdout=%dB stderr=%dB)", elapsed, summary, len(stdoutStr), len(stderrStr))
			}
		}
	}
	return stdoutStr, stderrStr, err
}

// waitForCommandWithContext waits for cmd to finish; on ctx cancellation it kills the process and waits for exit.
func waitForCommandWithContext(ctx context.Context, cmd *exec.Cmd) error {
	waitErrCh := make(chan error, 1)
	go func() {
		waitErrCh <- cmd.Wait()
	}()

	select {
	case err := <-waitErrCh:
		return err
	case <-ctx.Done():
		killVirtctlCmd(cmd)
		return waitForProcessExitAfterKill(ctx, waitErrCh)
	}
}

// waitForProcessExitAfterKill waits for cmd.Wait after a kill, bounded by killWait, or returns ctx.Err().
func waitForProcessExitAfterKill(ctx context.Context, waitErrCh <-chan error) error {
	const killWait = 5 * time.Second
	select {
	case waitErr := <-waitErrCh:
		if waitErr != nil {
			return fmt.Errorf("%w (process terminated: %v)", ctx.Err(), waitErr)
		}
		return ctx.Err()
	case <-time.After(killWait):
		return fmt.Errorf("%w (process did not exit within %s after kill)", ctx.Err(), killWait)
	}
}

// formatDeadlineRemaining formats the time left until ctx's deadline, or "none" if there is no deadline.
func formatDeadlineRemaining(ctx context.Context) string {
	deadline, ok := ctx.Deadline()
	if !ok {
		return "none"
	}
	remaining := time.Until(deadline).Round(time.Second)
	if remaining < 0 {
		remaining = 0
	}
	return remaining.String()
}

// formatRemoteCommandStreamsForInlineLog formats stdout and stderr for multi-line completion logs.
// Large outputs are truncated to the first and last inlineLogMaxHeadTailLines lines.
func formatRemoteCommandStreamsForInlineLog(stdout, stderr string) string {
	var b strings.Builder
	stderr = strings.TrimSpace(stderr)
	stdout = strings.TrimSpace(stdout)
	if stderr != "" {
		b.WriteString("stderr:\n")
		b.WriteString(stderr)
	}
	if stdout != "" {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("stdout:\n")
		b.WriteString(truncateMiddleLines(stdout, inlineLogMaxHeadTailLines))
	}
	if b.Len() == 0 {
		return "output: <empty stdout/stderr>"
	}
	return b.String()
}

// truncateMiddleLines keeps the first and last n lines, replacing the middle with a marker.
func truncateMiddleLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= 2*n {
		return s
	}
	head := lines[:n]
	tail := lines[len(lines)-n:]
	omitted := len(lines) - 2*n
	return strings.Join(head, "\n") +
		fmt.Sprintf("\n\n... (%d lines truncated) ...\n\n", omitted) +
		strings.Join(tail, "\n")
}

// SCPTo copies a local file to the guest using `virtctl scp`.
func (v Virtctl) SCPTo(ctx context.Context, namespace, vm, src, dst string) (stderr string, err error) {
	args := buildVirtctlSCPToArgs(v.Path, namespace, vm, v.IdentityFile, v.Username, v.knownHostsFile(), src, dst)
	_, stderrStr, err := v.run(ctx, args)
	return stderrStr, err
}

// buildVirtctlSCPToArgs builds the full argument list for `virtctl scp` uploading src to dst on the guest.
func buildVirtctlSCPToArgs(virtctlPath, namespace, vm, identityFile, username, knownHostsFile, src, dst string) []string {
	args := []string{
		virtctlPath, "scp",
		"--namespace", namespace,
		"--identity-file", identityFile,
		"--known-hosts", knownHostsFile,
	}
	args = appendLocalSSHOpts(args, knownHostsFile)
	if username != "" {
		args = append(args, "--username", username)
	}
	args = append(args, src, fmt.Sprintf("%s:%s", normalizeVirtctlTarget(vm), dst))
	return args
}

// --- Shared helpers used across virtctl.go and virtctl_ssh.go ---

// normalizeVirtctlTarget returns vm unchanged if it already includes a resource prefix; otherwise prefixes with "vmi/".
func normalizeVirtctlTarget(vm string) string {
	vm = strings.TrimSpace(vm)
	if strings.Contains(vm, "/") {
		return vm
	}
	return "vmi/" + vm
}

// appendLocalSSHOpts appends defaultLocalSSHOpts plus a dynamic UserKnownHostsFile to args as --local-ssh-opts pairs.
func appendLocalSSHOpts(args []string, knownHostsFile string) []string {
	for _, opt := range defaultLocalSSHOpts {
		args = append(args, "--local-ssh-opts", opt)
	}
	args = append(args, "--local-ssh-opts", "-o UserKnownHostsFile="+knownHostsFile)
	return args
}

// summarizeVirtctlCommand returns a short, log-safe description of args (subcommand-specific for ssh/scp).
func summarizeVirtctlCommand(args []string) string {
	if len(args) < 2 {
		return "virtctl <unknown>"
	}
	subcommand := args[1]
	switch subcommand {
	case "ssh":
		return summarizeVirtctlSSHCommand(args)
	case "scp":
		target := "<unknown target>"
		if pos := virtctlPositionalArgs(args); len(pos) >= 2 {
			target = pos[len(pos)-1]
		}
		return fmt.Sprintf("virtctl scp %s", target)
	default:
		return fmt.Sprintf("virtctl %s", subcommand)
	}
}

// summarizeVirtctlSSHCommand returns a short log line for a virtctl ssh argv (target and remote command summary).
func summarizeVirtctlSSHCommand(args []string) string {
	target := "<unknown target>"
	if pos := virtctlPositionalArgs(args); len(pos) > 0 {
		target = pos[0]
	}
	for i := range len(args) {
		if args[i] != "--command" || i+1 >= len(args) {
			continue
		}
		return fmt.Sprintf("virtctl ssh %s command=%s", target, strings.TrimSpace(args[i+1]))
	}
	return fmt.Sprintf("virtctl ssh %s", target)
}

// virtctlSSHVMTarget returns the vmi/vm target from virtctl ssh positionals, if present.
func virtctlSSHVMTarget(args []string) (string, bool) {
	if len(args) < 2 || args[1] != "ssh" {
		return "", false
	}
	pos := virtctlPositionalArgs(args)
	if len(pos) == 0 {
		return "", false
	}
	return pos[0], true
}

// virtctlFlagConsumesValue reports whether flag expects a separate value argument on the virtctl command line.
func virtctlFlagConsumesValue(flag string) bool {
	switch flag {
	case "--namespace", "--identity-file", "--known-hosts", "--local-ssh-opts", "--username", "--command":
		return true
	default:
		return false
	}
}

// virtctlPositionalArgs returns non-flag arguments from a virtctl argv (skipping the binary and subcommand).
func virtctlPositionalArgs(args []string) []string {
	if len(args) <= 2 {
		return nil
	}
	var positionals []string
	for i := 2; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--") {
			if virtctlFlagConsumesValue(arg) && i+1 < len(args) {
				i++
			}
			continue
		}
		positionals = append(positionals, arg)
	}
	return positionals
}
