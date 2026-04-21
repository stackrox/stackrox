//go:build !unix

package vmhelpers

import "os/exec"

// configureVirtctlCmdForCancellation is a no-op on non-Unix platforms (no separate process group).
func configureVirtctlCmdForCancellation(cmd *exec.Cmd) {}

// killVirtctlCmd terminates the virtctl process on non-Unix platforms.
func killVirtctlCmd(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
}
