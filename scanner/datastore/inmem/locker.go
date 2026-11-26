package inmem

import (
	"context"
	"sync"

	"github.com/quay/claircore/libvuln"
	"github.com/quay/claircore/libvuln/updates"
)

// Compile-time assertions that Locker satisfies both ClairCore lock interfaces.
var (
	_ libvuln.LockSource = (*Locker)(nil)
	_ updates.LockSource = (*Locker)(nil)
)

// Locker is a process-local, keyed lock source.
//
// Each in-memory matcher pod owns its own address space and must import bundles
// independently rather than sharing a distributed database lock (which would
// cause replicas to skip importing bundles locked by a peer).
type Locker struct {
	mu    sync.Mutex
	locks map[string]chan struct{}
}

// NewLocker returns an empty in-process Locker.
func NewLocker() *Locker {
	return &Locker{
		locks: make(map[string]chan struct{}),
	}
}

func (l *Locker) getChan(key string) chan struct{} {
	l.mu.Lock()
	defer l.mu.Unlock()
	ch, ok := l.locks[key]
	if !ok {
		ch = make(chan struct{}, 1)
		l.locks[key] = ch
	}
	return ch
}

// TryLock attempts to acquire key without blocking. If key is already held or
// parent is already canceled, it returns a canceled context and a no-op unlock.
func (l *Locker) TryLock(parent context.Context, key string) (context.Context, context.CancelFunc) {
	if parent.Err() != nil {
		return parent, func() {}
	}
	ch := l.getChan(key)
	select {
	case ch <- struct{}{}:
		ctx, cancel := context.WithCancel(parent)
		var once sync.Once
		return ctx, func() {
			once.Do(func() {
				cancel()
				<-ch
			})
		}
	default:
		ctx, cancel := context.WithCancel(parent)
		cancel()
		return ctx, cancel
	}
}

// Lock acquires key, blocking until it is available or parent is canceled.
func (l *Locker) Lock(parent context.Context, key string) (context.Context, context.CancelFunc) {
	if parent.Err() != nil {
		return parent, func() {}
	}
	ch := l.getChan(key)
	select {
	case ch <- struct{}{}:
		ctx, cancel := context.WithCancel(parent)
		var once sync.Once
		return ctx, func() {
			once.Do(func() {
				cancel()
				<-ch
			})
		}
	case <-parent.Done():
		return parent, func() {}
	}
}

// Close satisfies libvuln.LockSource.
func (l *Locker) Close(_ context.Context) error {
	return nil
}
