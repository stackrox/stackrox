// +build !release

package sync

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/debug"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	// TODO(ROX-2163) - tracks the move to 5 seconds from 1 second
	defaultLockTimeout = 5 * time.Second

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
	utils.Must(errors.Wrap(err, "could not parse watchdog timeout setting"))
	lockTimeout = time.Duration(timeoutSecs) * time.Second
}

func watchdog(action string, ch <-chan struct{}, timeout time.Duration, stacktrace debug.LazyStacktrace) {
	t := time.NewTimer(timeout)

	select {
	case <-ch:
		if !t.Stop() {
			<-t.C
		}
		return
	case <-t.C:
		panic(fmt.Errorf("Action %s took more than %v to complete. Stack trace:\n%s", action, timeout, stacktrace))
	}
}

func panicOnTimeout(action string, do func(), timeout time.Duration) {
	ch := make(chan struct{})
	go watchdog(action, ch, timeout, debug.GetLazyStacktrace(2))
	do()
	close(ch)
}

// Mutex is a watchdog-enabled version of sync.Mutex.
type Mutex struct {
	sync.Mutex
}

// Lock acquires the lock on the mutex.
func (m *Mutex) Lock() {
	panicOnTimeout("Mutex.Lock", m.Mutex.Lock, lockTimeout)
}

// RWMutex is a watchdog-enabled version of sync.RWMutex.
type RWMutex struct {
	sync.RWMutex
}

// RLock acquires a reader lock on the mutex.
func (m *RWMutex) RLock() {
	panicOnTimeout("RWMutex.RLock", m.RWMutex.RLock, lockTimeout)
}

// Lock acquires a writer (exclusive) lock on the mutex.
func (m *RWMutex) Lock() {
	panicOnTimeout("RWMutex.Lock", m.RWMutex.Lock, lockTimeout)
}
