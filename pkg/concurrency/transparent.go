package concurrency

import (
	"sync/atomic"
)

// A TransparentMutex is a mutex which allows callers to
// perform a non-blocking lock (that returns a bool indicating whether it succeeded).
// The zero-value is ready to use.
type TransparentMutex struct {
	locked int32
}

// MaybeLock tries to lock, and returns a bool indicating whether the lock was acquired.
// The caller MUST check the return value, and
// - not do anything requiring synchronization if the value is false.
// - unlock the mutex eventually if the value is true (
func (t *TransparentMutex) MaybeLock() bool {
	return atomic.CompareAndSwapInt32(&t.locked, 0, 1)
}

// Unlock unlocks the TransparentMutex. The caller must NOT call unlock unless it knows it holds the lock, else the
// behaviour is undefined.
func (t *TransparentMutex) Unlock() {
	atomic.StoreInt32(&t.locked, 0)
}
