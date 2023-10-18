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

// WithRLock1 locks the RLocker of the given lockable, executes `do`, returns T, and releases the lock.
// This function is panic-safe, i.e., even if `do()` panics, the lock will be released.
func WithRLock1[T any](locker RLocker, do func() T) T {
	locker.RLock()
	defer locker.RUnlock()

	return do()
}

// WithRLock2 locks the RLocker of the given lockable, executes `do`, returns T and E, and releases the lock.
// This function is panic-safe, i.e., even if `do()` panics, the lock will be released.
func WithRLock2[T any, E any](locker RLocker, do func() (T, E)) (T, E) {
	locker.RLock()
	defer locker.RUnlock()

	return do()
}

// WithLock1 locks the given locker, executes `do`, returns T, and releases the lock.
// This function is panic-safe, i.e., even if `do()` panics, the lock will be released.
func WithLock1[T any](locker sync.Locker, do func() T) T {
	locker.Lock()
	defer locker.Unlock()

	return do()
}

// WithLock2 locks the given locker, executes `do`, returns T and E, and releases the lock.
// This function is panic-safe, i.e., even if `do()` panics, the lock will be released.
func WithLock2[T any, E any](locker sync.Locker, do func() (T, E)) (T, E) {
	locker.Lock()
	defer locker.Unlock()

	return do()
}

// WithLock3 locks the given locker, executes `do`, returns T, S , E, and releases the lock.
// This function is panic-safe, i.e., even if `do()` panics, the lock will be released.
func WithLock3[T any, E any, S any](locker sync.Locker, do func() (T, E, S)) (T, E, S) {
	locker.Lock()
	defer locker.Unlock()

	return do()
}
