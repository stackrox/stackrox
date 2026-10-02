package lifecycle

import (
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

// Gate serializes cluster deletion with writes that must not outlive the cluster.
type Gate struct {
	mu      sync.Mutex
	entries map[string]*entry
}

type entry struct {
	mu       sync.RWMutex
	stateMu  sync.Mutex
	deleting bool
	refs     int
}

var singleton = New()

// Singleton returns the process-wide cluster lifecycle gate.
func Singleton() *Gate {
	return singleton
}

// New creates a cluster lifecycle gate.
func New() *Gate {
	return &Gate{entries: make(map[string]*entry)}
}

// Enter acquires a shared lifecycle lease unless deletion has started. The
// returned release function must be called when active is true.
func (g *Gate) Enter(clusterID string) (active bool, release func()) {
	e := g.retain(clusterID)

	active = concurrency.WithLock1(&e.stateMu, func() bool {
		if e.deleting {
			return false
		}
		e.mu.RLock()
		return true
	})
	if !active {
		g.release(clusterID, e)
		return false, nil
	}

	return true, func() {
		concurrency.UnsafeRUnlock(&e.mu)
		g.release(clusterID, e)
	}
}

// BeginDeletion prevents new active writes and waits for in-flight writes to finish.
// The returned function must be called once cluster cleanup completes.
func (g *Gate) BeginDeletion(clusterID string) func() {
	e := g.retain(clusterID)
	concurrency.WithLock(&e.stateMu, func() {
		e.deleting = true
	})
	// Drain in-flight read locks, then release. Cleanup can exceed the dev
	// mutex watchdog (10s), which aborts the process on a long exclusive hold.
	concurrency.WithLock(&e.mu, func() {})

	return func() {
		concurrency.WithLock(&e.stateMu, func() {
			e.deleting = false
		})
		g.release(clusterID, e)
	}
}

func (g *Gate) retain(clusterID string) *entry {
	g.mu.Lock()
	defer g.mu.Unlock()
	e := g.entries[clusterID]
	if e == nil {
		e = &entry{}
		g.entries[clusterID] = e
	}
	e.refs++
	return e
}

func (g *Gate) release(clusterID string, e *entry) {
	g.mu.Lock()
	defer g.mu.Unlock()
	e.refs--
	if e.refs == 0 && !e.deleting {
		delete(g.entries, clusterID)
	}
}
