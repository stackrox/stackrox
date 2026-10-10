package backgroundworker

import "context"

// Startable is any worker that can be started with a context.
type Startable interface {
	Start(ctx context.Context)
}

// Stoppable is any worker that can be stopped.
type Stoppable interface {
	Stop()
}

// Worker is a background worker with full lifecycle control.
type Worker interface {
	Startable
	Stoppable
	Registerable
}

// WorkerGroup coordinates Start and Stop for a set of workers. Start
// launches all workers; Stop shuts them down in reverse order (last
// started is first stopped), which is the natural dependency ordering
// for most multi-worker decompositions.
type WorkerGroup struct {
	workers []Worker
}

// NewWorkerGroup creates a group from the given workers. Start and Stop
// order follows the slice order (Stop is reversed).
func NewWorkerGroup(workers ...Worker) *WorkerGroup {
	return &WorkerGroup{workers: workers}
}

// Start starts all workers in order with the given context.
func (g *WorkerGroup) Start(ctx context.Context) {
	for _, w := range g.workers {
		w.Start(ctx)
	}
}

// Stop stops all workers in reverse order and blocks until each has
// exited before stopping the next.
func (g *WorkerGroup) Stop() {
	for i := len(g.workers) - 1; i >= 0; i-- {
		g.workers[i].Stop()
	}
}

// Register registers all workers in the group with the given registry.
func (g *WorkerGroup) Register(r *Registry) {
	for _, w := range g.workers {
		r.Register(w)
	}
}
