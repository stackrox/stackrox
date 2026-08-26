package reportqueue

import (
	"context"

	reportGen "github.com/stackrox/rox/central/reports/scheduler/v2/reportgenerator"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/queue"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

// ReportQueue is a thread-safe FIFO queue of report requests. It tracks which
// report configurations are currently running so that at most one report per
// config executes at a time, and stores cancel functions for running reports
// to support user-initiated cancellation.
type ReportQueue struct {
	mu                sync.Mutex
	queue             *queue.Queue[*reportGen.ReportRequest]
	runningConfigs    set.StringSet
	jobIDToCancelFunc map[string]context.CancelCauseFunc
}

// New creates a new empty ReportQueue.
func New() *ReportQueue {
	return &ReportQueue{
		queue:             queue.NewQueue[*reportGen.ReportRequest](),
		runningConfigs:    set.NewStringSet(),
		jobIDToCancelFunc: make(map[string]context.CancelCauseFunc),
	}
}

// Enqueue adds a report request to the back of the queue.
// The underlying queue is internally synchronized, so no additional locking is needed.
func (q *ReportQueue) Enqueue(req *reportGen.ReportRequest) {
	q.queue.Push(req)
}

// Dequeue removes and returns the first runnable report request from the queue.
// VIEW_BASED reports are always runnable. Config-based reports are runnable only
// if their config ID is not in the running set. Returns nil if no runnable
// request is found. For non-VIEW_BASED reports, the config ID is added to the
// running set upon dequeue.
func (q *ReportQueue) Dequeue() *reportGen.ReportRequest {
	q.mu.Lock()
	defer q.mu.Unlock()

	req, ok := q.queue.PullWithPred(func(req *reportGen.ReportRequest) bool {
		if req.ReportSnapshot.GetReportStatus().GetReportRequestType() == storage.ReportStatus_VIEW_BASED {
			return true
		}
		return !q.runningConfigs.Contains(req.ReportSnapshot.GetReportConfigurationId())
	})
	if !ok {
		return nil
	}
	if req.ReportSnapshot.GetReportStatus().GetReportRequestType() != storage.ReportStatus_VIEW_BASED {
		q.runningConfigs.Add(req.ReportSnapshot.GetReportConfigurationId())
	}
	return req
}

// Remove removes a queued (not yet running) request by report snapshot ID.
// Returns the removed request, or nil if not found.
// The underlying queue is internally synchronized, so no additional locking is needed.
func (q *ReportQueue) Remove(reportID string) *reportGen.ReportRequest {
	req, _ := q.queue.PullWithPred(func(req *reportGen.ReportRequest) bool {
		return req.ReportSnapshot.GetReportId() == reportID
	})
	return req
}

// MarkReportDoneForConfig removes a config ID from the running set, allowing
// another report for the same config to be dequeued.
func (q *ReportQueue) MarkReportDoneForConfig(configID string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.runningConfigs.Remove(configID)
}

// AddCancelFunc stores a cancel function for a running report, keyed by report ID.
func (q *ReportQueue) AddCancelFunc(reportID string, cancel context.CancelCauseFunc) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.jobIDToCancelFunc[reportID] = cancel
}

// RemoveCancelFunc removes the cancel function for the given report ID.
func (q *ReportQueue) RemoveCancelFunc(reportID string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.jobIDToCancelFunc, reportID)
}

// TryCancel calls the cancel function for the given report ID with
// ErrUserCancelled as the cause. Returns true if a cancel function was found
// and invoked.
func (q *ReportQueue) TryCancel(reportID string) bool {
	cancel := concurrency.WithLock1(&q.mu, func() context.CancelCauseFunc {
		return q.jobIDToCancelFunc[reportID]
	})
	if cancel == nil {
		return false
	}
	cancel(reportGen.ErrUserCancelled)
	return true
}

// Len returns the number of items in the queue.
// The underlying queue is internally synchronized, so no additional locking is needed.
func (q *ReportQueue) Len() int {
	return q.queue.Len()
}
