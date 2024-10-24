//go:build !release

package sync

import (
	"fmt"
	"sync"
	"time"
)

type DaveRWMutex struct {
	sync.RWMutex
	acquireTime time.Time // Lock only, not RLock
}

// RLock acquires a reader lock on the mutex.
func (m *DaveRWMutex) RLock(trace string, span string) {
	name := fmt.Sprintf("DaveRWMutex.RLock (trace %q, span %q)", trace, span)
	panicOnTimeout(name, m.RWMutex.RLock, lockTimeout)
}

// Lock acquires a writer (exclusive) lock on the mutex.
func (m *DaveRWMutex) Lock(trace string, span string) {
	name := fmt.Sprintf("DaveRWMutex.Lock (trace %q, span %q)", trace, span)
	panicOnTimeout(name, m.RWMutex.Lock, lockTimeout)
	m.acquireTime = time.Now()
}

// TryRLock wraps the call to sync.RWMutex TryRLock. It returns true if the lock was acquired.
func (m *DaveRWMutex) TryRLock() bool {
	if m.RWMutex.TryRLock() {
		m.acquireTime = time.Now()
		return true
	}
	return false
}

// TryLock wraps the call to sync.RWMutex TryLock. It returns true if the lock was acquired.
func (m *DaveRWMutex) TryLock() bool {
	if m.RWMutex.TryLock() {
		m.acquireTime = time.Now()
		return true
	}
	return false
}

// Unlock releases an acquired writer (exclusive) lock on the mutex.
func (m *DaveRWMutex) Unlock(trace string, span string) {
	name := fmt.Sprintf("DaveRWMutex.Unlock (trace %q, span %q)", trace, span)
	panicIfTooMuchTimeElapsed(name, m.acquireTime, lockTimeout, 1)
	defer m.RWMutex.Unlock() // suppress the roxvet error for calling Unlock()
}
