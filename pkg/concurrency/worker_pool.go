package concurrency

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/sync"
	"golang.org/x/sync/semaphore"
)

var (
	ctx = context.Background()
)

// WorkerPool will allow running concurrent jobs with an upperbound on concurrency.
// A worker pool is a pool of available routines that can run jobs. The number of available routines is controlled by
// maxConcurrency field. A job is a func() that will run in a go routine. To request a job, use the AddJob(job func()) method.
// If more than maxConcurrency jobs are requested, the caller to AddJob(job func()) will be blocked until another running job finishes execution.
// A WorkerPool must be initialized is using the NewWorkerPool(maxConcurrency int) function that will set the necessary state to run concurrent jobs.
// If WorkerPool is instantiated using struct instantiation, using any of the methods will lead to undefined behavior.
// After initialization, use Start() method to start the pool. When the worker pool is no longer needed, use Stop() method
// to stop the pool from waiting for jobs. If the pool is not stopped, it will keep waiting for jobs to run.
type WorkerPool struct {
	maxConcurrency int
	jobs           chan func()
	jobsSema       *semaphore.Weighted
	stopper        Stopper
	waitGroup      sync.WaitGroup
	started        bool
	stopped        bool
}

func NewWorkerPool(maxConcurrency int) *WorkerPool {
	return &WorkerPool{
		maxConcurrency: maxConcurrency,
		jobs:           make(chan func(), maxConcurrency),
		jobsSema:       semaphore.NewWeighted(int64(maxConcurrency)),
		stopper:        NewStopper(),
		started:        false,
		stopped:        false,
	}
}

// Start worker pool. The worker pool will start waiting for jobs for this. Can only be started once per instance.
// Cannot be restarted if stopped.
func (w *WorkerPool) Start() error {
	if w.stopped {
		return errors.New("Worker pool is already stopped. It cannot be re-started.")
	}
	if w.started {
		return errors.New("Worker pool was already started.")
	}
	w.started = true
	for i := 0; i < w.maxConcurrency; i++ {
		w.waitGroup.Add(1)
		go w.jobRunner()
	}
	return nil
}

func (w *WorkerPool) jobRunner() {
	defer w.waitGroup.Done()
	for {
		select {
		case <-w.stopper.Flow().StopRequested():
			return
		case job := <-w.jobs:
			w.runJob(job)
		}
	}
}

func (w *WorkerPool) runJob(task func()) {
	defer w.jobsSema.Release(1)
	task()
}

// GetMaxConcurrency returns the maximum number of jobs that can be concurrently run by the worker pool
func (w *WorkerPool) GetMaxConcurrency() int {
	return w.maxConcurrency
}

// AddJob will run a new job. It will block the caller if maxConcurrency jobs are already running
func (w *WorkerPool) AddJob(job func()) error {
	if w.stopped || !w.started {
		return errors.New("Worker pool is no longer running")
	}
	// Will block if maxConcurrency jobs are already running
	err := w.jobsSema.Acquire(ctx, 1)
	if err != nil {
		return err
	}
	w.jobs <- job
	return nil
}

// Stop will make the worker pool to stop waiting for jobs. Any currently running jobs will be completed before it returns.
// Worker pool cannot be restarted once stopped.
func (w *WorkerPool) Stop() error {
	if !w.started {
		return errors.New("Worker pool was never started.")
	}
	w.stopped = true
	w.stopper.Client().Stop()
	w.waitGroup.Wait()
	return nil
}
