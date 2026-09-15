package backgroundworker

import (
	"sync"
)

// Registerable is implemented by any worker that can report its status. It
// is satisfied by *PeriodicWorker and any future worker archetypes that
// expose a Status method with this signature.
type Registerable interface {
	Status() WorkerStatus
}

// Registry tracks all background workers for observability. It is safe for
// concurrent use.
type Registry struct {
	mu      sync.RWMutex
	workers []Registerable
}

// NewRegistry constructs an empty Registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds w to the registry. Registered workers are included in
// subsequent calls to All and in the DebugHandler output.
func (r *Registry) Register(w Registerable) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.workers = append(r.workers, w)
}

// All returns a snapshot of the current status of every registered worker,
// in registration order.
func (r *Registry) All() []WorkerStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	statuses := make([]WorkerStatus, 0, len(r.workers))
	for _, w := range r.workers {
		statuses = append(statuses, w.Status())
	}
	return statuses
}

// Global is the process-wide registry. Workers register here for
// discoverability via the debug endpoint.
var Global = NewRegistry()
