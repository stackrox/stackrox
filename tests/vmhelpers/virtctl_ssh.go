package vmhelpers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// SSH runs `virtctl ssh` against the VM and returns captured streams.
func (v Virtctl) SSH(ctx context.Context, namespace, vm string, command ...string) (stdout string, stderr string, err error) {
	args := v.buildSSHArgs(namespace, vm, command...)
	return v.run(ctx, args)
}

// buildSSHArgs builds the full argv for `virtctl ssh`, optionally with a quoted remote --command.
func (v Virtctl) buildSSHArgs(namespace, vm string, command ...string) []string {
	args := []string{
		v.Path, "ssh",
		"--namespace", namespace,
		"--identity-file", v.IdentityFile,
		"--known-hosts", v.KnownHostsFile,
	}
	args = v.appendLocalSSHOpts(args)
	if v.Username != "" {
		args = append(args, "--username", v.Username)
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
	args := v.buildSCPToArgs(namespace, vm, src, dst)
	_, stderrStr, err := v.run(ctx, args)
	return stderrStr, err
}

// buildSCPToArgs builds the full argument list for `virtctl scp` uploading src to dst on the guest.
func (v Virtctl) buildSCPToArgs(namespace, vm, src, dst string) []string {
	args := []string{
		v.Path, "scp",
		"--namespace", namespace,
		"--identity-file", v.IdentityFile,
		"--known-hosts", v.KnownHostsFile,
	}
	args = v.appendLocalSSHOpts(args)
	if v.Username != "" {
		args = append(args, "--username", v.Username)
	}
	args = append(args, src, fmt.Sprintf("%s:%s", normalizeVirtctlTarget(vm), dst))
	return args
}
