package vmhelpers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// defaultLocalSSHOpts are SSH client options passed on every virtctl ssh/scp invocation via --local-ssh-opts.
var defaultLocalSSHOpts = []string{
	"-o StrictHostKeyChecking=no",
	"-o UserKnownHostsFile=/dev/null",
	"-o IdentitiesOnly=yes",
	"-o ConnectTimeout=5",
}

// Constants for virtctl remote logging: heartbeat interval and max length for inlined command output.
const (
	defaultVirtctlHeartbeatInterval = 30 * time.Second
)

// normalizeVirtctlTarget returns vm unchanged if it already includes a resource prefix; otherwise prefixes with "vmi/".
func normalizeVirtctlTarget(vm string) string {
	vm = strings.TrimSpace(vm)
	if strings.Contains(vm, "/") {
		return vm
	}
	return "vmi/" + vm
}

// appendDefaultLocalSSHOpts appends defaultLocalSSHOpts to args as repeated --local-ssh-opts pairs.
func appendDefaultLocalSSHOpts(args []string) []string {
	for _, opt := range defaultLocalSSHOpts {
		args = append(args, "--local-ssh-opts", opt)
	}
	return args
}

// buildVirtctlSCPToArgs builds the full argument list for `virtctl scp` uploading src to dst on the guest.
func buildVirtctlSCPToArgs(virtctlPath, namespace, vm, identityFile, username, src, dst string) []string {
	args := []string{
		virtctlPath, "scp",
		"--namespace", namespace,
		"--identity-file", identityFile,
		"--known-hosts", "/dev/null",
	}
	args = appendDefaultLocalSSHOpts(args)
	if username != "" {
		args = append(args, "--username", username)
	}
	args = append(args, src, fmt.Sprintf("%s:%s", normalizeVirtctlTarget(vm), dst))
	return args
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

const inlineLogMaxHeadTailLines = 100

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

// Virtctl runs virtctl subcommands with optional per-call timeout.
type Virtctl struct {
	Path           string
	IdentityFile   string
	Username       string
	CommandTimeout time.Duration
	// Logf is optional. When provided, each remote command logs start/heartbeat/completion.
	Logf func(format string, args ...any)
	// HeartbeatInterval controls "still running" log cadence for long commands.
	// Zero uses defaultVirtctlHeartbeatInterval.
	HeartbeatInterval time.Duration
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
		if v.Logf != nil {
			v.Logf("remote command failed to start: %s (err=%v)", summary, err)
		}
		return outBuf.String(), errBuf.String(), err
	}
	err = waitForCommandWithContext(ctx, cmd)

	if stopHeartbeat != nil {
		close(stopHeartbeat)
		hbWG.Wait()
	}
	if v.Logf != nil {
		elapsed := time.Since(start).Round(time.Second)
		if err != nil {
			v.Logf("remote command failed in %s: %s (outcome=%v stdout=%dB stderr=%dB)\n%s",
				elapsed, summary, err, outBuf.Len(), errBuf.Len(), formatRemoteCommandStreamsForInlineLog(outBuf.String(), errBuf.String()))
		} else {
			v.Logf("remote command complete in %s: %s (stdout=%dB stderr=%dB)\n%s",
				elapsed, summary, outBuf.Len(), errBuf.Len(), formatRemoteCommandStreamsForInlineLog(outBuf.String(), errBuf.String()))
		}
	}
	return outBuf.String(), errBuf.String(), err
}

// SCPTo copies a local file to the guest using `virtctl scp`.
func (v Virtctl) SCPTo(ctx context.Context, namespace, vm, src, dst string) (stderr string, err error) {
	args := buildVirtctlSCPToArgs(v.Path, namespace, vm, v.IdentityFile, v.Username, src, dst)
	_, stderrStr, err := v.run(ctx, args)
	return stderrStr, err
}
