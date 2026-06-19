//go:build test

package vmhelpers

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSSHCommandArgs_UsesIdentityAndNamespace(t *testing.T) {
	t.Parallel()

	virt := Virtctl{
		Path:           "/usr/bin/virtctl",
		IdentityFile:   "/tmp/id_rsa",
		Username:       "cloud-user",
		KnownHostsFile: "/dev/null",
	}
	args := virt.buildSSHArgs("stackrox", "vm-rhel9", "sudo", "true")
	require.Equal(t, []string{
		"/usr/bin/virtctl", "ssh",
		"--namespace", "stackrox",
		"--identity-file", "/tmp/id_rsa",
		"--known-hosts", "/dev/null",
		"--local-ssh-opts", "-o StrictHostKeyChecking=no",
		"--local-ssh-opts", "-o IdentitiesOnly=yes",
		"--local-ssh-opts", "-o ConnectTimeout=5",
		"--local-ssh-opts", "-o UserKnownHostsFile=/dev/null",
		"--username", "cloud-user",
		"vmi/vm-rhel9",
		"--command", `"sudo" "true"`,
	}, args)
}

func TestBuildVirtctlSSHCommand_QuotesArguments(t *testing.T) {
	t.Parallel()

	got := buildVirtctlSSHCommand("sh", "-c", `echo "hello world" && true`)
	require.Equal(t, `"sh" "-c" "echo \"hello world\" && true"`, got)
}

func TestSCPToArgs_RemoteTargetShape(t *testing.T) {
	t.Parallel()

	virt := Virtctl{
		Path:           "/usr/bin/virtctl",
		IdentityFile:   "/tmp/id_rsa",
		Username:       "cloud-user",
		KnownHostsFile: "/dev/null",
	}
	args := virt.buildSCPToArgs("stackrox", "vm-rhel9", "/local/roxagent", "/usr/local/bin/roxagent")
	require.Equal(t, []string{
		"/usr/bin/virtctl", "scp",
		"--namespace", "stackrox",
		"--identity-file", "/tmp/id_rsa",
		"--known-hosts", "/dev/null",
		"--local-ssh-opts", "-o StrictHostKeyChecking=no",
		"--local-ssh-opts", "-o IdentitiesOnly=yes",
		"--local-ssh-opts", "-o ConnectTimeout=5",
		"--local-ssh-opts", "-o UserKnownHostsFile=/dev/null",
		"--username", "cloud-user",
		"/local/roxagent", "vmi/vm-rhel9:/usr/local/bin/roxagent",
	}, args)
}

func TestSummarizeVirtctlCommand_SSHWithRemoteCommand(t *testing.T) {
	t.Parallel()

	virt := Virtctl{
		Path:           "/usr/bin/virtctl",
		IdentityFile:   "/tmp/id_rsa",
		Username:       "cloud-user",
		KnownHostsFile: "/dev/null",
	}
	args := virt.buildSSHArgs("stackrox", "vm-rhel9", "sudo", "true")
	require.Equal(t, `virtctl ssh vmi/vm-rhel9 command="sudo" "true"`, summarizeVirtctlCommand(args))
}

func TestSummarizeVirtctlCommand_SCP(t *testing.T) {
	t.Parallel()

	virt := Virtctl{
		Path:           "/usr/bin/virtctl",
		IdentityFile:   "/tmp/id_rsa",
		Username:       "cloud-user",
		KnownHostsFile: "/dev/null",
	}
	args := virt.buildSCPToArgs("stackrox", "vm-rhel9", "/local/roxagent", "/usr/local/bin/roxagent")
	require.Equal(t, "virtctl scp vmi/vm-rhel9:/usr/local/bin/roxagent", summarizeVirtctlCommand(args))
}

func TestVirtctlRun_RespectsContextDeadline(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	virt := Virtctl{
		Logf: func(string, ...any) {},
	}
	_, _, err := virt.run(ctx, []string{"sh", "-c", "sleep 5"})
	require.Error(t, err)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Less(t, time.Since(start), 3*time.Second, "virtctl run should terminate promptly on context deadline")
}

func TestVirtctlRunLogsStreamsOnSuccess(t *testing.T) {
	t.Parallel()

	var logs []string
	virt := Virtctl{
		Logf: func(format string, args ...any) {
			logs = append(logs, formatMessage(format, args...))
		},
	}

	stdout, stderr, err := virt.run(context.Background(), []string{
		"/bin/sh", "-c", "printf 'stdout-line\\n'; printf 'stderr-line\\n' >&2",
	})

	require.NoError(t, err)
	require.Equal(t, "stdout-line\n", stdout)
	require.Equal(t, "stderr-line\n", stderr)
	require.NotEmpty(t, logs)

	lastLog := logs[len(logs)-1]
	require.Contains(t, lastLog, "remote command complete")
	require.Contains(t, lastLog, "stdout:\nstdout-line")
	require.Contains(t, lastLog, "stderr:\nstderr-line")
}

func TestVirtctlRun_PanicsWithoutLogf(t *testing.T) {
	t.Parallel()

	require.Panics(t, func() {
		_, _, _ = (Virtctl{}).run(context.Background(), []string{
			"/bin/sh", "-c", "printf 'stdout-line\\n'; printf 'stderr-line\\n' >&2",
		})
	})
}

func TestFormatRemoteCommandStreamsForInlineLogTruncatesStdout(t *testing.T) {
	t.Parallel()

	var stdoutLines []string
	for i := range 2*inlineLogMaxHeadTailLines + 1 {
		stdoutLines = append(stdoutLines, fmt.Sprintf("stdout-line-%03d", i))
	}

	formatted := formatRemoteCommandStreamsForInlineLog(strings.Join(stdoutLines, "\n"), "")

	require.Contains(t, formatted, "stdout:\nstdout-line-000")
	require.Contains(t, formatted, "... (1 lines truncated) ...")
	require.Contains(t, formatted, "stdout-line-200")
}

func formatMessage(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}
