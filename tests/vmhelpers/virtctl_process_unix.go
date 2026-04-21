//go:build unix

package vmhelpers

import (
	"os/exec"
	"syscall"
)

// configureVirtctlCmdForCancellation puts the child in its own process group so signals target the whole tree.
func configureVirtctlCmdForCancellation(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killVirtctlCmd sends SIGKILL to the virtctl process group, then kills the main process.
func killVirtctlCmd(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	if pid := cmd.Process.Pid; pid > 0 {
		// Kill the whole process group so child ssh/scp processes cannot outlive virtctl.
		_ = syscall.Kill(-pid, syscall.SIGKILL)
	}
	_ = cmd.Process.Kill()
}
