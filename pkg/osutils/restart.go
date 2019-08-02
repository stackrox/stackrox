package osutils

import (
	"fmt"
	"os"
	"syscall"
)

// Restart restarts the program.
func Restart() {
	exePath := os.Getenv("RESTART_EXE")
	var err error
	if exePath == "" {
		exePath, err = os.Executable()
		if err != nil {
			exePath = "/proc/self/exe"
		}
	}
	err = syscall.Exec(exePath, os.Args, os.Environ())
	_, _ = fmt.Fprintf(os.Stderr, "Exec-based restarting of %s failed: %v. Restarting via exit; you might see CrashLoopBackoff states as a result", exePath, err)
	os.Exit(1)
}
