package postgres

import (
	"context"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/notify"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	cacheSetupTimeout      = 10 * time.Second
	cacheRefreshTimeout    = 30 * time.Second
	cacheReconcileInterval = time.Minute
	cacheRetryInterval     = time.Second
	cacheBatchDelay        = 20 * time.Millisecond
	cachePendingKeys       = 1024
	cachePendingBytes      = 256 * 1024
)

// WithCacheCoordination enables retained caches to follow all committed table
// writes. Reusing this wrapper shares one listener across all store instances.
// Close cancels and joins its work before closing the underlying database.
// The database owner must call Close outside listener/observer callbacks.
// WithoutStoreCache takes precedence, in either wrapper order.
func WithCacheCoordination(ctx context.Context, db postgres.DB) postgres.DB {
	if storeCacheDisabled(db) || cacheCoordinatorFor(db) != nil {
		return db
	}
	ctx, cancel := context.WithCancel(ctx)
	return &cacheCoordinator{
		DB: db, ctx: ctx, cancel: cancel, ready: make(chan struct{}),
		tables: make(map[string]*cacheTable),
	}
}

type cacheTable struct {
	ready    chan struct{}
	relation uint32
	err      error
	stores   []*cacheRegistration
}

type cacheCoordinator struct {
	postgres.DB
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.Mutex
	tables    map[string]*cacheTable
	connected bool
	closed    bool
	ready     chan struct{}
	start     sync.Once
	close     sync.Once
	workers   sync.WaitGroup
}

func (c *cacheCoordinator) Unwrap() postgres.DB { return c.DB }

func cacheCoordinatorFor(db postgres.DB) *cacheCoordinator {
	for db != nil {
		if coordinator, ok := db.(*cacheCoordinator); ok {
			return coordinator
		}
		wrapper, ok := db.(interface{ Unwrap() postgres.DB })
		if !ok {
			return nil
		}
		db = wrapper.Unwrap()
	}
	return nil
}

func (c *cacheCoordinator) Close() {
	c.close.Do(func() {
		concurrency.WithLock(&c.mu, func() {
			c.closed = true
			c.cancel()
		})
		c.workers.Wait()
		c.DB.Close()
	})
}

func (c *cacheCoordinator) isConnected() bool {
	return concurrency.WithLock1(&c.mu, func() bool { return c.connected && c.ctx.Err() == nil })
}

// No database work or store callbacks are performed under the registry lock.
func (c *cacheCoordinator) connectionChanged(connected bool) {
	stores := concurrency.WithLock1(&c.mu, func() []*cacheRegistration {
		c.connected = connected
		var stores []*cacheRegistration
		for _, table := range c.tables {
			stores = append(stores, table.stores...)
		}
		return stores
	})
	for _, store := range stores {
		store.invalidate()
		store.enqueue(nil, true)
	}
}

func (c *cacheCoordinator) register(ctx context.Context, tableName string, r *cacheRegistration) error {
	// Add before starting any work so Close cannot race WaitGroup.Add.
	if err := concurrency.WithLock1(&c.mu, func() error {
		if c.closed || c.ctx.Err() != nil {
			return context.Canceled
		}
		c.workers.Add(1)
		c.start.Do(func() {
			c.workers.Go(func() {
				var ready sync.Once
				listener := notify.NewListenerWithHooks(c.DB, c.notification, notify.LifecycleHooks{
					AfterListen: func(context.Context) error {
						c.connectionChanged(true)
						ready.Do(func() { close(c.ready) })
						return nil
					},
					OnDisconnect: func(err error) { c.connectionChanged(false) },
				}, cacheNotificationChannel)
				listener.Listen(c.ctx)
			})
		})
		return nil
	}); err != nil {
		return err
	}
	// The registration's worker owns this count after successful setup.
	started := false
	defer func() {
		if !started {
			c.workers.Done()
		}
	}()
	select {
	case <-c.ready:
	case <-ctx.Done():
		return ctx.Err()
	}
	var install bool
	table := concurrency.WithLock1(&c.mu, func() *cacheTable {
		if table := c.tables[tableName]; table != nil {
			return table
		}
		install = true
		table := &cacheTable{ready: make(chan struct{})}
		c.tables[tableName] = table
		return table
	})
	if install {
		table.relation, table.err = installCacheNotifications(ctx, c.DB, tableName)
		close(table.ready)
	}
	select {
	case <-table.ready:
	case <-ctx.Done():
		return ctx.Err()
	}
	if table.err != nil {
		concurrency.WithLock(&c.mu, func() {
			if c.tables[tableName] == table {
				delete(c.tables, tableName)
			}
		})
		return table.err
	}
	concurrency.WithLock(&c.mu, func() { table.stores = append(table.stores, r) })
	// Events are already queued for this store. The initial scan must not
	// consume those events: they may describe commits after its snapshot.
	if err := r.refresh(ctx, nil, true); err != nil {
		r.invalidate()
		r.enqueue(nil, true)
		log.Errorf("Initial coordinated cache refresh for %s: %v", tableName, err)
	}
	r.signal()
	started = true
	go func() {
		defer c.workers.Done()
		r.run(c.ctx)
	}()
	return nil
}

// Each registration has a bounded queue and its own worker so an observer or
// query failure cannot hold up another instance, including of the same table.
type cacheRegistration struct {
	coordinator *cacheCoordinator
	mu          sync.Mutex
	keys        map[string]struct{}
	bytes       int
	full        bool
	wake        chan struct{}
	invalidate  func()
	refresh     func(context.Context, []string, bool) error
	deliver     func(context.Context) error
}

func (r *cacheRegistration) signal() {
	select {
	case r.wake <- struct{}{}:
	default:
	}
}

func (r *cacheRegistration) enqueue(keys []string, full bool) {
	overflow := concurrency.WithLock1(&r.mu, func() bool {
		r.full = r.full || full
		var overflow bool
		if !r.full {
			for _, key := range keys {
				if _, exists := r.keys[key]; exists {
					continue
				}
				if len(r.keys) >= cachePendingKeys || r.bytes+len(key) > cachePendingBytes {
					r.full = true
					overflow = true
					break
				}
				r.keys[key] = struct{}{}
				r.bytes += len(key)
			}
		}
		if r.full {
			clear(r.keys)
			r.bytes = 0
		}
		return overflow
	})
	if overflow {
		r.invalidate()
	}
	r.signal()
}

func (r *cacheRegistration) take() ([]string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	keys := make([]string, 0, len(r.keys))
	for key := range r.keys {
		keys = append(keys, key)
	}
	full := r.full
	clear(r.keys)
	r.bytes, r.full = 0, false
	return keys, full
}

func (r *cacheRegistration) run(ctx context.Context) {
	ticker := time.NewTicker(cacheReconcileInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.enqueue(nil, true)
		case <-r.wake:
		}
		if !cacheWait(ctx, cacheBatchDelay) {
			return
		}
		if !r.coordinator.isConnected() {
			continue
		}
		if err := r.process(ctx); err != nil {
			log.Errorf("Coordinated cache refresh/delivery: %v", err)
			r.invalidate()
			r.enqueue(nil, true)
			if !cacheWait(ctx, cacheRetryInterval) {
				return
			}
		}
	}
}

func (r *cacheRegistration) process(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, cacheRefreshTimeout)
	defer cancel()
	// Retry retained observer deliveries before applying another refresh.
	if err := r.deliver(ctx); err != nil {
		return err
	}
	keys, full := r.take()
	if len(keys) != 0 || full {
		if err := r.refresh(ctx, keys, full); err != nil {
			return err
		}
	}
	return r.deliver(ctx)
}

func cacheWait(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
