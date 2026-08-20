package reportqueue

import (
	"container/list"
	"context"

	reportGen "github.com/stackrox/rox/central/reports/scheduler/v2/reportgenerator"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

// ReportQueue is a thread-safe FIFO queue of report requests. It tracks which
// report configurations are currently running so that at most one report per
// config executes at a time, and stores cancel functions for running reports
// to support user-initiated cancellation.
type ReportQueue struct {
	mu                sync.Mutex
	queue             *list.List
	runningConfigs    set.StringSet
	jobIDToCancelFunc map[string]context.CancelCauseFunc
}

// New creates a new empty ReportQueue.
func New() *ReportQueue {
	return &ReportQueue{
		queue:             list.New(),
		runningConfigs:    set.NewStringSet(),
		jobIDToCancelFunc: make(map[string]context.CancelCauseFunc),
	}
}

// Enqueue adds a report request to the back of the queue.
func (q *ReportQueue) Enqueue(req *reportGen.ReportRequest) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.queue.PushBack(req)
}

// Dequeue removes and returns the first runnable report request from the queue.
// VIEW_BASED reports are always runnable. Config-based reports are runnable only
// if their config ID is not in the running set. Returns nil if no runnable
// request is found. For non-VIEW_BASED reports, the config ID is added to the
// running set upon dequeue.
func (q *ReportQueue) Dequeue() *reportGen.ReportRequest {
	q.mu.Lock()
	defer q.mu.Unlock()

	req := findAndRemove(q.queue, func(req *reportGen.ReportRequest) bool {
		if req.ReportSnapshot.GetReportStatus().GetReportRequestType() == storage.ReportStatus_VIEW_BASED {
			return true
		}
		return !q.runningConfigs.Contains(req.ReportSnapshot.GetReportConfigurationId())
	})
	if req == nil {
		return nil
	}
	if req.ReportSnapshot.GetReportStatus().GetReportRequestType() != storage.ReportStatus_VIEW_BASED {
		q.runningConfigs.Add(req.ReportSnapshot.GetReportConfigurationId())
	}
	return req
}

// Remove removes a queued (not yet running) request by report snapshot ID.
// Returns the removed request, or nil if not found.
func (q *ReportQueue) Remove(reportID string) *reportGen.ReportRequest {
	q.mu.Lock()
	defer q.mu.Unlock()

	return findAndRemove(q.queue, func(req *reportGen.ReportRequest) bool {
		return req.ReportSnapshot.GetReportId() == reportID
	})
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
func (q *ReportQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.queue.Len()
}

// findAndRemove walks the queue front-to-back and removes the first element
// matching the predicate. Returns the removed request, or nil if none matched.
func findAndRemove(queue *list.List, pred func(req *reportGen.ReportRequest) bool) *reportGen.ReportRequest {
	cur := queue.Front()
	for cur != nil {
		req, ok := cur.Value.(*reportGen.ReportRequest)
		if ok && pred(req) {
			return queue.Remove(cur).(*reportGen.ReportRequest)
		}
		cur = cur.Next()
	}
	return nil
}
