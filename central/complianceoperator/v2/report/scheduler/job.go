package scheduler

import (
	"context"

	"github.com/stackrox/rox/pkg/queue"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

// reportJob is a submitted report of one scan configuration.
type reportJob struct {
	scanConfigID   string
	scanConfigName string
}

// jobQueue is a FIFO queue of report jobs that ensures at most one job per scan configuration
// runs at a time: a job whose configuration is already running stays queued until the running
// job completes and the scheduler is re-signaled.
type jobQueue struct {
	mu             sync.Mutex
	queue          *queue.Queue[*reportJob]
	runningConfigs set.StringSet
}

func newJobQueue() *jobQueue {
	return &jobQueue{
		queue:          queue.NewQueue[*reportJob](),
		runningConfigs: set.NewStringSet(),
	}
}

// enqueue adds the job to the back of the queue. The underlying queue is internally
// synchronized, so no additional locking is needed.
func (q *jobQueue) enqueue(job *reportJob) {
	q.queue.Push(job)
}

// dequeue removes and returns the first job whose configuration is not running, marking that
// configuration as running. It returns nil when no job is runnable.
func (q *jobQueue) dequeue() *reportJob {
	q.mu.Lock()
	defer q.mu.Unlock()
	job, ok := q.queue.PullWithPred(func(job *reportJob) bool {
		return !q.runningConfigs.Contains(job.scanConfigID)
	})
	if !ok {
		return nil
	}
	q.runningConfigs.Add(job.scanConfigID)
	return job
}

// markDone removes the configuration from the running set, allowing its next queued job to be
// dequeued.
func (q *jobQueue) markDone(scanConfigID string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.runningConfigs.Remove(scanConfigID)
}

// runReportJob is the report job of a scan configuration.
//
// Iteration 1 of the decoupled reporting rollout: a no-op placeholder. Later iterations settle
// running scans, audit expectations, and generate and deliver the report.
func runReportJob(_ context.Context, job *reportJob) {
	reportJobsFired.WithLabelValues(job.scanConfigName).Inc()
	log.Infof("Compliance report due for scan configuration %q", job.scanConfigName)
}
