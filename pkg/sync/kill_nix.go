//go:build !release && (linux || darwin)

package sync

import "syscall"

func kill() {
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGABRT); err != nil {
		go func() { panic(err) }() // do this in a Goroutine to prevent any deferred `recover()` from catching it.
	}
}
