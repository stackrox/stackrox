package lock

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

// DefaultLockPath is the default path for the roxagent singleton lock file.
// /run/lock is a tmpfs directory (per FHS) that the kernel clears on reboot,
// ensuring no stale locks persist after crashes. The Quadlet container unit
// bind-mounts this path so the lock coordinates host-binary and container runs.
const DefaultLockPath = "/run/lock/roxagent/roxagent.lock"

// Result is the outcome of TryLock.
type Result int

const (
	// Acquired means the lock was taken; the release function must be called when done.
	Acquired Result = iota
	// Held means another process already holds the lock (non-blocking flock would block).
	Held
	// Unavailable means the lock could not be used (filesystem or permission error).
	Unavailable
)

// TryLock attempts a non-blocking exclusive flock on path.
// The parent directory is created with mode 0755 if missing. The lock file is created with 0600 if needed.
//
// On Acquired, release is non-nil and closes the file descriptor (releasing the flock).
// On Held, release and err are nil.
// On Unavailable, release is nil and err describes the failure.
func TryLock(path string) (Result, func(), error) {
	if path == "" {
		return Unavailable, nil, errors.New("lock path is empty")
	}

	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return Unavailable, nil, err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return Unavailable, nil, err
	}

	// On Linux, EAGAIN and EWOULDBLOCK are the same errno for flock(2) with LOCK_NB,
	// but we check both for clarity and resilience.
	// Explanation: LOCK_EX is the exclusive lock, and LOCK_NB is the non-blocking flag.
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return Held, nil, nil
		}
		return Unavailable, nil, err
	}

	release := func() {
		// Explicitly unlock before closing. Closing the fd alone releases the flock on Linux,
		// but issuing LOCK_UN first is a defensive practice adopted by many projects to avoid
		// relying solely on fd-close semantics (e.g. if the fd were accidentally dup'd).
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}
	return Acquired, release, nil
}

// RunWithLock acquires the lock, runs fn, and releases.
// onHeld is called when another process holds the lock.
// onUnavailable is called when the lock cannot be used (permissions, missing mount, etc.).
// If onHeld or onUnavailable return a non-nil error, RunWithLock returns that error.
// If they return nil, RunWithLock returns nil (the caller chose to skip or degrade gracefully).
func RunWithLock(path string, fn func() error, onHeld func() error, onUnavailable func(error) error) error {
	res, release, lockErr := TryLock(path)
	switch res {
	case Acquired:
		defer release()
		return fn()
	case Held:
		return onHeld()
	case Unavailable:
		return onUnavailable(lockErr)
	default:
		return errors.New("unexpected lock result")
	}
}
