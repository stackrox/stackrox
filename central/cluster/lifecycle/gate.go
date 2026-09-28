package lifecycle

import "sync"

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

	e.stateMu.Lock()
	if e.deleting {
		e.stateMu.Unlock()
		g.release(clusterID, e)
		return false, nil
	}
	e.mu.RLock()
	e.stateMu.Unlock()

	return true, func() {
		e.mu.RUnlock()
		g.release(clusterID, e)
	}
}

// BeginDeletion prevents new active writes and waits for existing writes to finish.
// The returned function must be called once cluster cleanup completes.
func (g *Gate) BeginDeletion(clusterID string) func() {
	e := g.retain(clusterID)
	e.stateMu.Lock()
	e.deleting = true
	e.stateMu.Unlock()
	e.mu.Lock()

	return func() {
		e.mu.Unlock()
		e.stateMu.Lock()
		e.deleting = false
		e.stateMu.Unlock()
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
