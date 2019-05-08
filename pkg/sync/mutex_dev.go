// +build !release

package sync

import (
	"fmt"
	"sync"
	"time"

	"github.com/stackrox/rox/pkg/debug"
)

const (
	// TODO(ROX-2163) - tracks the move to 5 seconds from 1 second
	lockTimeout = 5 * time.Second
)

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
