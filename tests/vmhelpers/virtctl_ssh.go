package vmhelpers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// SSH runs `virtctl ssh` against the VM and returns captured streams.
func (v Virtctl) SSH(ctx context.Context, namespace, vm string, command ...string) (stdout string, stderr string, err error) {
	args := buildVirtctlSSHArgs(v.Path, namespace, vm, v.IdentityFile, v.Username, v.KnownHostsFile, command...)
	return v.run(ctx, args)
}

// buildVirtctlSSHArgs builds the full argv for `virtctl ssh`, optionally with a quoted remote --command.
func buildVirtctlSSHArgs(virtctlPath, namespace, vm, identityFile, username, knownHostsFile string, command ...string) []string {
	args := []string{
		virtctlPath, "ssh",
		"--namespace", namespace,
		"--identity-file", identityFile,
		"--known-hosts", knownHostsFile,
	}
	args = appendLocalSSHOpts(args, knownHostsFile)
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
	quoted := make([]string, len(command))
	for i, arg := range command {
		quoted[i] = strconv.Quote(arg)
	}
	return strings.Join(quoted, " ")
}

// SCPTo copies a local file to the guest using `virtctl scp`.
func (v Virtctl) SCPTo(ctx context.Context, namespace, vm, src, dst string) (stderr string, err error) {
	args := buildVirtctlSCPToArgs(v.Path, namespace, vm, v.IdentityFile, v.Username, v.KnownHostsFile, src, dst)
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
