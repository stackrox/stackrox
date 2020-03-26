// +build !release
// +build linux darwin

package sync

import "syscall"

func kill() {
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGABRT); err != nil {
		panic(err)
	}
}
