//go:build !linux
package memlimit

// setMemoryLimit is a no-op for non-Linux environments.
func setMemoryLimit() int64 {
	return 0
}
