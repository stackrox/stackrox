package concurrency

import "github.com/stackrox/rox/pkg/sync"

// RLocker is an interface for objects that allow locking for read access only (such as `sync.RWMutex`).
type RLocker interface {
	RLock()
	RUnlock()
}

// WithLock locks the given locker, executes `do`, and releases the lock. This function is panic-safe, i.e., even if
// `do()` panics, the lock will be released.
func WithLock(locker sync.Locker, do func()) {
	locker.Lock()
	defer locker.Unlock()

	do()
}

// WithRLock locks the RLocker of the given lockable, executes `do`, and releases the lock. This function is panic-safe,
// i.e., even if `do()` panics, the lock will be released.
func WithRLock(locker RLocker, do func()) {
	locker.RLock()
	defer locker.RUnlock()

	do()
}
