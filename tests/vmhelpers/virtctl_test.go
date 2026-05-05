//go:build test

package vmhelpers

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSSHCommandArgs_UsesIdentityAndNamespace(t *testing.T) {
	args := buildVirtctlSSHArgs("/usr/bin/virtctl", "stackrox", "vm-rhel9", "/tmp/id_rsa", "cloud-user", "sudo", "true")
	require.Equal(t, []string{
		"/usr/bin/virtctl", "ssh",
		"--namespace", "stackrox",
		"--identity-file", "/tmp/id_rsa",
		"--known-hosts", "/dev/null",
		"--local-ssh-opts", "-o StrictHostKeyChecking=no",
		"--local-ssh-opts", "-o UserKnownHostsFile=/dev/null",
		"--local-ssh-opts", "-o IdentitiesOnly=yes",
		"--local-ssh-opts", "-o ConnectTimeout=5",
		"--username", "cloud-user",
		"vmi/vm-rhel9",
		"--command", `"sudo" "true"`,
	}, args)
}

func TestBuildVirtctlSSHCommand_QuotesArguments(t *testing.T) {
	got := buildVirtctlSSHCommand("sh", "-c", `echo "hello world" && true`)
	require.Equal(t, `"sh" "-c" "echo \"hello world\" && true"`, got)
}

func TestSCPToArgs_RemoteTargetShape(t *testing.T) {
	args := buildVirtctlSCPToArgs("/usr/bin/virtctl", "stackrox", "vm-rhel9", "/tmp/id_rsa", "cloud-user", "/local/roxagent", "/usr/local/bin/roxagent")
	require.Equal(t, []string{
		"/usr/bin/virtctl", "scp",
		"--namespace", "stackrox",
		"--identity-file", "/tmp/id_rsa",
		"--known-hosts", "/dev/null",
		"--local-ssh-opts", "-o StrictHostKeyChecking=no",
		"--local-ssh-opts", "-o UserKnownHostsFile=/dev/null",
		"--local-ssh-opts", "-o IdentitiesOnly=yes",
		"--local-ssh-opts", "-o ConnectTimeout=5",
		"--username", "cloud-user",
		"/local/roxagent", "vmi/vm-rhel9:/usr/local/bin/roxagent",
	}, args)
}

func TestSummarizeVirtctlCommand_SSHWithRemoteCommand(t *testing.T) {
	args := buildVirtctlSSHArgs("/usr/bin/virtctl", "stackrox", "vm-rhel9", "/tmp/id_rsa", "cloud-user", "sudo", "true")
	require.Equal(t, `virtctl ssh vmi/vm-rhel9 command="sudo" "true"`, summarizeVirtctlCommand(args))
}

func TestSummarizeVirtctlCommand_SCP(t *testing.T) {
	args := buildVirtctlSCPToArgs("/usr/bin/virtctl", "stackrox", "vm-rhel9", "/tmp/id_rsa", "cloud-user", "/local/roxagent", "/usr/local/bin/roxagent")
	require.Equal(t, "virtctl scp vmi/vm-rhel9:/usr/local/bin/roxagent", summarizeVirtctlCommand(args))
}

func TestVirtctlRun_RespectsContextDeadline(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, _, err := (Virtctl{}).run(ctx, []string{"sh", "-c", "sleep 5"})
	require.Error(t, err)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Less(t, time.Since(start), 3*time.Second, "virtctl run should terminate promptly on context deadline")
}

func TestSummarizeRemoteSSHCommand(t *testing.T) {
	t.Parallel()

	require.Equal(t, "<empty>", summarizeRemoteSSHCommand(" \n\t "))
	require.Equal(t, `"sudo" "true"`, summarizeRemoteSSHCommand(`"sudo" "true"`))

	long := strings.Repeat("x", 300)
	require.Equal(t, long, summarizeRemoteSSHCommand(long))
}
