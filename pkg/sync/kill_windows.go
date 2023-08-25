//go:build !release && windows

package sync

func kill() {
	panic("windows doesn't support syscall.Kill")
}
