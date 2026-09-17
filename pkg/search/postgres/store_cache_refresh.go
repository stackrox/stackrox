package postgres

import (
	"context"
	"fmt"
	"maps"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sac"
)

// CacheChanges describes a cache reconciliation. Deleted contains previous
// objects for derived cleanup. Full requests authoritative derived reconciliation,
// including on subscription. Objects belong to the callback and may be modified.
type CacheChanges[PT any] struct {
	Upserts []PT
	Deleted []PT
	Full    bool
}

// ObserveCache subscribes to a coordinated store through an optional typed
// capability; generated store interfaces need no observer methods. Deliveries
// are asynchronous, serial per store, and outside all internal locks. Callbacks
// should reread current storage under their own lock, since local writes can
// overtake a delivery. The callback context has all-access SAC scopes and follows
// the DB lifecycle and refresh timeout. Errors (and panics) retain the delivery
// for retry and make reads use PostgreSQL. Callbacks must respect cancellation. Unsubscribe prevents
// new calls but does not wait for an in-flight call (and is safe inside a call).
func ObserveCache[PT any](store any, fn func(context.Context, CacheChanges[PT]) error) (func(), error) {
	observer, ok := store.(interface {
		ObserveCache(func(context.Context, CacheChanges[PT]) error) (func(), error)
	})
	if !ok {
		return nil, errors.New("store does not support cache observation")
	}
	if fn == nil {
		return nil, errors.New("cache observer must not be nil")
	}
	return observer.ObserveCache(fn)
}

func (c *cachedStore[T, PT]) initializeCache(db postgres.DB) error {
	coordinator := cacheCoordinatorFor(db)
	if coordinator == nil {
		return c.populateCache()
	}
	if c.registration == nil {
		c.registration = &cacheRegistration{
			coordinator: coordinator, keys: make(map[string]struct{}), wake: make(chan struct{}, 1),
			invalidate: c.invalidateCache, refresh: c.refreshCache, deliver: c.deliverCacheChanges,
		}
	}
	ctx, cancel := context.WithTimeout(coordinator.ctx, cacheSetupTimeout)
	defer cancel()
	return coordinator.register(ctx, c.schema.Table, c.registration)
}

func (c *cachedStore[T, PT]) retryCacheInitialization(db postgres.DB) {
	coordinator := cacheCoordinatorFor(db)
	if coordinator == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(cacheRetryInterval)
		defer ticker.Stop()
		for {
			select {
			case <-coordinator.ctx.Done():
				return
			case <-ticker.C:
				if err := c.initializeCache(db); err == nil {
					return
				} else {
					log.Errorf("Retrying coordinated store cache initialization: %v", err)
				}
			}
		}
	}()
}

func (c *cachedStore[T, PT]) invalidateCache() {
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	c.generation++
	c.trusted = false
}

func (c *cachedStore[T, PT]) useCache(ctx context.Context) bool {
	return !postgres.HasTxInContext(ctx) && c.CacheEnabled()
}

// beginMutation covers the entire DB operation and cache application interval.
// Caller-owned transactions bypass the gate: they may already own row locks
// needed by an ordinary writer holding the gate. They never publish tentative
// state. A second invalidation prevents an overlapping scan restoring trust.
func (c *cachedStore[T, PT]) beginMutation(ctx context.Context) func() {
	if tx, ok := postgres.TxFromContext(ctx); ok {
		c.pendingTransactions.Add(1)
		c.invalidateCache()
		tx.AfterFinish(func() {
			c.pendingTransactions.Add(-1)
			if c.registration != nil {
				c.registration.enqueue(nil, true)
			}
		})
		return c.invalidateCache
	}
	c.mutationGate.RLock()
	return c.mutationGate.RUnlock
}

func (c *cachedStore[T, PT]) mutationFailed() {
	c.invalidateCache()
	if c.registration != nil {
		c.registration.enqueue(nil, true)
	}
}

func (c *cachedStore[T, PT]) signalObservers() {
	if c.registration != nil {
		c.registration.signal()
	}
}

// Local cache changes pause once an observer backlog overflows or the store is
// otherwise untrusted. Leave the old map intact for the next full scan's diff;
// reads already use PostgreSQL. This bounds tombstones even if a callback is
// slow while local writes continue to commit.
func (c *cachedStore[T, PT]) removeFromCacheNoLock(id string) {
	if c.registration != nil && !c.trusted {
		return
	}
	if previous := c.cache[id]; previous != nil && c.observerCount.Load() != 0 {
		if len(c.removed) >= cachePendingKeys {
			c.generation++
			c.trusted = false
			c.registration.enqueue(nil, true)
			return
		}
		c.removed = append(c.removed, previous)
	}
	delete(c.cache, id)
}

// refreshCache builds a complete result separately. No partial query or decode
// failure can erase rows. Only a full scan can restore trust; its generation
// must still match so a newer disconnect/overflow/transaction is not forgotten.
func (c *cachedStore[T, PT]) refreshCache(ctx context.Context, keys []string, full bool) error {
	c.mutationGate.Lock()
	defer c.mutationGate.Unlock()
	generation := concurrency.WithRLock1(&c.cacheLock, func() uint64 { return c.generation })
	ctx = sac.WithAllAccess(ctx)
	next := make(map[string]PT)
	var missing []int
	if full {
		if err := c.underlyingStore.Walk(ctx, func(obj PT) error {
			next[c.pkGetter(obj)] = obj.CloneVT()
			return nil
		}); err != nil {
			c.invalidateCache()
			return err
		}
	} else {
		objects, misses, err := c.underlyingStore.GetMany(ctx, keys)
		if err != nil {
			c.invalidateCache()
			return err
		}
		missing = misses
		for _, obj := range objects {
			next[c.pkGetter(obj)] = obj.CloneVT()
		}
	}
	connected := c.registration == nil || c.registration.coordinator.isConnected()
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	if full {
		if c.observerCount.Load() != 0 {
			for key, previous := range c.cache {
				if _, exists := next[key]; !exists {
					c.removed = append(c.removed, previous)
				}
			}
		}
		c.cache = next
		c.fullVersion++
		c.trusted = c.generation == generation && connected && c.pendingTransactions.Load() == 0
		if !c.trusted {
			return errors.New("cache invalidated during full reconciliation")
		}
	} else {
		maps.Copy(c.cache, next)
		for _, index := range missing {
			c.removeFromCacheNoLock(keys[index])
		}
	}
	c.setCacheEntriesGauge()
	return nil
}

// Only the registration worker touches an observer's delivered/pending state.
// The two snapshots retain immutable references, bounded by the store size even
// while callbacks fail and local writes continue. In particular, deleted objects
// survive a successful generic refresh followed by a failed derived callback.
type cacheObserver[PT any] struct {
	fn          func(context.Context, CacheChanges[PT]) error
	active      atomic.Bool
	delivered   map[string]PT
	fullVersion uint64
	pending     *cacheDelivery[PT]
}

type cacheDelivery[PT any] struct {
	changes     CacheChanges[PT]
	next        map[string]PT
	fullVersion uint64
}

func (c *cachedStore[T, PT]) ObserveCache(fn func(context.Context, CacheChanges[PT]) error) (func(), error) {
	if c.registration == nil || fn == nil {
		return nil, errors.New("cache observation requires a coordinated store and callback")
	}
	if err := c.registration.coordinator.ctx.Err(); err != nil {
		return nil, err
	}
	observer := &cacheObserver[PT]{fn: fn}
	observer.active.Store(true)
	concurrency.WithLock(&c.observerLock, func() {
		if c.observers == nil {
			c.observers = make(map[*cacheObserver[PT]]struct{})
		}
		c.observers[observer] = struct{}{}
		c.observerCount.Add(1)
	})
	c.registration.signal()
	return func() {
		if !observer.active.Swap(false) {
			return
		}
		c.observerCount.Add(-1)
		concurrency.WithLock(&c.observerLock, func() { delete(c.observers, observer) })
	}, nil
}

func (c *cachedStore[T, PT]) deliverCacheChanges(ctx context.Context) error {
	ctx = sac.WithAllAccess(ctx)
	observers := concurrency.WithLock1(&c.observerLock, func() []*cacheObserver[PT] {
		observers := make([]*cacheObserver[PT], 0, len(c.observers))
		for observer := range c.observers {
			observers = append(observers, observer)
		}
		return observers
	})
	// Complete every retained delivery before draining new tombstones. Failed
	// callbacks must not turn the bounded local queue into an unbounded retry
	// backlog, nor cause another observer to lose the same deletion.
	if err := c.deliverPending(ctx, observers); err != nil {
		return err
	}
	if len(observers) == 0 {
		concurrency.WithLock(&c.cacheLock, func() { c.removed = nil })
		return nil
	}
	if !c.CacheEnabled() {
		return nil
	}
	var next map[string]PT
	var removed []PT
	var fullVersion uint64
	concurrency.WithLock(&c.cacheLock, func() {
		next, removed, fullVersion = maps.Clone(c.cache), c.removed, c.fullVersion
		c.removed = nil
	})
	// Prepare for every observer before invoking external code. All share the
	// immutable snapshot, while each callback receives its own object clones.
	for _, observer := range observers {
		if !observer.active.Load() {
			continue
		}
		changes := CacheChanges[PT]{Full: observer.delivered == nil || observer.fullVersion != fullVersion}
		for key, obj := range next {
			if changes.Full || obj != observer.delivered[key] {
				changes.Upserts = append(changes.Upserts, obj)
			}
		}
		deleted := make(map[PT]struct{}, len(removed))
		for _, obj := range removed {
			deleted[obj] = struct{}{}
		}
		for obj := range deleted {
			changes.Deleted = append(changes.Deleted, obj)
		}
		observer.pending = &cacheDelivery[PT]{changes: changes, next: next, fullVersion: fullVersion}
	}
	return c.deliverPending(ctx, observers)
}

func (c *cachedStore[T, PT]) deliverPending(ctx context.Context, observers []*cacheObserver[PT]) error {
	for _, observer := range observers {
		pending := observer.pending
		if !observer.active.Load() || pending == nil {
			continue
		}
		if pending.changes.Full || len(pending.changes.Upserts) != 0 || len(pending.changes.Deleted) != 0 {
			if err := c.callObserver(ctx, observer.fn, pending.changes); err != nil {
				return err
			}
		}
		observer.delivered, observer.fullVersion = pending.next, pending.fullVersion
		observer.pending = nil
	}
	return nil
}

func (c *cachedStore[T, PT]) callObserver(ctx context.Context, fn func(context.Context, CacheChanges[PT]) error, changes CacheChanges[PT]) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("cache observer panicked: %v", recovered)
		}
	}()
	copy := CacheChanges[PT]{Full: changes.Full, Upserts: make([]PT, 0, len(changes.Upserts)), Deleted: make([]PT, 0, len(changes.Deleted))}
	for _, obj := range changes.Upserts {
		copy.Upserts = append(copy.Upserts, obj.CloneVT())
	}
	for _, obj := range changes.Deleted {
		copy.Deleted = append(copy.Deleted, obj.CloneVT())
	}
	return fn(ctx, copy)
}
