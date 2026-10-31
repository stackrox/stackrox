package backgroundworker

import (
	"sync"
)

// Registerable is implemented by any worker that can report its status.
// Satisfied by PeriodicWorker, QueueConsumer, BatchAccumulator, and
// RunOnceWorker.
type Registerable interface {
	Status() WorkerStatus
}

// Registry tracks all background workers for observability.
type Registry struct {
	mu      sync.RWMutex
	workers []Registerable
}

// NewRegistry constructs an empty Registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds w to the registry.
func (r *Registry) Register(w Registerable) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.workers = append(r.workers, w)
}

// Deregister removes w from the registry. Use for per-connection workers
// whose lifetime is shorter than the process. No-op if w is not registered.
func (r *Registry) Deregister(w Registerable) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, existing := range r.workers {
		if existing == w {
			r.workers = append(r.workers[:i], r.workers[i+1:]...)
			return
		}
	}
}

// All returns the current status of every registered worker.
func (r *Registry) All() []WorkerStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	statuses := make([]WorkerStatus, 0, len(r.workers))
	for _, w := range r.workers {
		statuses = append(statuses, w.Status())
	}
	return statuses
}

// Global is the process-wide registry.
var Global = NewRegistry()
