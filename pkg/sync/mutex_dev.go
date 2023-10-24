//go:build !release

package sync

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/debug"
)

const (
	defaultLockTimeout = 10 * time.Second

	// We omit the `ROX_` prefix as this is fairly general (potential open-sourcing). Also, this is not a setting in the
	// classical sense that is read by roxctl and propagated to deploying. It is a debug setting that normally should
	// not be changed (and if it does, it will usually happen via editing the deployment directly, in any case with a
	// small impact radius).
	lockTimeoutSettingEnvVar = "MUTEX_WATCHDOG_TIMEOUT_SECS"
)

var (
	lockTimeout = defaultLockTimeout
)

func init() {
	// Deliberately initialize lockTimeout in an init function instead of via a sync.Once to keep the overhead
	// when creating mutexes as low as possible.
	timeoutSettingStr := os.Getenv(lockTimeoutSettingEnvVar)
	if timeoutSettingStr == "" {
		return
	}
	timeoutSecs, err := strconv.Atoi(timeoutSettingStr)
	if err != nil {
		panic(errors.Wrap(err, "could not parse watchdog timeout setting"))
	}
	lockTimeout = time.Duration(timeoutSecs) * time.Second
}

func panicIfTooMuchTimeElapsed(action string, startTime time.Time, limit time.Duration, skip int) {
	if limit <= 0 || time.Since(startTime) <= limit {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, "Action %s took more than %v to complete. Stack trace:\n%s", action, limit, debug.GetLazyStacktrace(skip+1))
	kill()
}

func panicOnTimeout(action string, do func(), timeout time.Duration) {
	panicOnTimeoutMarked(action, do, timeout, time.Now().UnixNano())
}

// panicOnTimeoutMarked allows recording the timestamp (in nanoseconds since unix epoch) as a parameter on the
// stack. The noinline directive is supposed to prevent the optimizer from removing it.
//
//go:noinline
func panicOnTimeoutMarked(action string, do func(), timeout time.Duration, nowNanos int64) {
	do()
	panicIfTooMuchTimeElapsed(action, time.Unix(0, nowNanos), timeout, 3)
}

// Mutex is a watchdog-enabled version of sync.Mutex.
type Mutex struct {
	sync.Mutex
	acquireTime time.Time
}

// Lock acquires the lock on the mutex.
func (m *Mutex) Lock() {
	panicOnTimeout("Mutex.Lock", m.Mutex.Lock, lockTimeout)
	m.acquireTime = time.Now()
}

// TryLock wraps the call to sync.Mutex TryLock. It returns true if the lock was acquired.
func (m *Mutex) TryLock() bool {
	if m.Mutex.TryLock() {
		m.acquireTime = time.Now()
		return true
	}
	return false
}

// Unlock releases an acquired lock on the mutex.
func (m *Mutex) Unlock() {
	panicIfTooMuchTimeElapsed("Mutex.Unlock", m.acquireTime, lockTimeout, 1)
	m.Mutex.Unlock()
}

// RWMutex is a watchdog-enabled version of sync.RWMutex.
type RWMutex struct {
	sync.RWMutex
	acquireTime time.Time // Lock only, not RLock
}

// RLock acquires a reader lock on the mutex.
func (m *RWMutex) RLock() {
	panicOnTimeout("RWMutex.RLock", m.RWMutex.RLock, lockTimeout)
}

// Lock acquires a writer (exclusive) lock on the mutex.
func (m *RWMutex) Lock() {
	panicOnTimeout("RWMutex.Lock", m.RWMutex.Lock, lockTimeout)
	m.acquireTime = time.Now()
}

// TryRLock wraps the call to sync.RWMutex TryRLock. It returns true if the lock was acquired.
func (m *RWMutex) TryRLock() bool {
	if m.RWMutex.TryRLock() {
		m.acquireTime = time.Now()
		return true
	}
	return false
}

// TryLock wraps the call to sync.RWMutex TryLock. It returns true if the lock was acquired.
func (m *RWMutex) TryLock() bool {
	if m.RWMutex.TryLock() {
		m.acquireTime = time.Now()
		return true
	}
	return false
}

// Unlock releases an acquired writer (exclusive) lock on the mutex.
func (m *RWMutex) Unlock() {
	panicIfTooMuchTimeElapsed("RWMutex.Unlock", m.acquireTime, lockTimeout, 1)
	m.RWMutex.Unlock()
}
