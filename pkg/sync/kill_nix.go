// +build !release
// +build linux darwin

package sync

import (
	"golang.org/x/sys/unix"
)

func kill() {
	if err := unix.Kill(unix.Getpid(), unix.SIGABRT); err != nil {
		panic(err)
	}
}
