package vmhelpers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// buildVirtctlSSHArgs builds the full argv for `virtctl ssh`, optionally with a quoted remote --command.
func buildVirtctlSSHArgs(virtctlPath, namespace, vm, identityFile, username string, command ...string) []string {
	args := []string{
		virtctlPath, "ssh",
		"--namespace", namespace,
		"--identity-file", identityFile,
		"--known-hosts", "/dev/null",
	}
	args = appendDefaultLocalSSHOpts(args)
	if username != "" {
		args = append(args, "--username", username)
	}
	args = append(args, normalizeVirtctlTarget(vm))
	if len(command) > 0 {
		args = append(args, "--command", buildVirtctlSSHCommand(command...))
	}
	return args
}

// buildVirtctlSSHCommand joins shell command parts into one string with per-argument strconv.Quote quoting.
func buildVirtctlSSHCommand(command ...string) string {
	if len(command) == 0 {
		return ""
	}
	quoted := make([]string, 0, len(command))
	for _, arg := range command {
		quoted = append(quoted, strconv.Quote(arg))
	}
	return strings.Join(quoted, " ")
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
		return fmt.Sprintf("virtctl ssh %s command=%s", target, summarizeRemoteSSHCommand(args[i+1]))
	}
	return fmt.Sprintf("virtctl ssh %s", target)
}

// summarizeRemoteSSHCommand returns the remote command string for the "start" log line (full --command value).
func summarizeRemoteSSHCommand(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return "<empty>"
	}
	return cmd
}

// SSH runs `virtctl ssh` against the VM and returns captured streams.
func (v Virtctl) SSH(ctx context.Context, namespace, vm string, command ...string) (stdout string, stderr string, err error) {
	args := buildVirtctlSSHArgs(v.Path, namespace, vm, v.IdentityFile, v.Username, command...)
	return v.run(ctx, args)
}
