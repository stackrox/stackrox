package scheduler

import (
	"time"

	"github.com/stackrox/rox/pkg/queue"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

// reportJob is a submitted report of one scan configuration.
type reportJob struct {
	scanConfigID   string
	scanConfigName string
	// cycleStart anchors the scan cycle the report belongs to: the expectation audit classifies
	// scans as fresh only if they started at or after it, and the settler waits only for its
	// running scans. Re-run and first-scan reports set it to the trigger time; scheduled fires
	// leave it zero and the anchor is resolved to the schedule tick that most recently passed
	// when the job runs.
	cycleStart time.Time
	// snapshotID is set for on-demand reports: the snapshot persisted at submission, which the
	// job picks up at execution. It is empty for scheduled and one-shot reports, which build
	// their snapshot at execution time.
	snapshotID string
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

// Len returns the number of queued jobs. The underlying queue is internally
// synchronized, so no additional locking is needed.
func (q *jobQueue) Len() int {
	return q.queue.Len()
}
