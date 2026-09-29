package reportqueue

import (
	"context"
	"testing"

	reportGen "github.com/stackrox/rox/central/reports/scheduler/v2/reportgenerator"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeRequest(reportID, configID string, requestType storage.ReportStatus_RunMethod) *reportGen.ReportRequest {
	return &reportGen.ReportRequest{
		ReportSnapshot: &storage.ReportSnapshot{
			ReportId:              reportID,
			ReportConfigurationId: configID,
			ReportStatus: &storage.ReportStatus{
				ReportRequestType: requestType,
			},
		},
	}
}

func TestEnqueueDequeueFIFO(t *testing.T) {
	q := New()

	req1 := makeRequest("r1", "c1", storage.ReportStatus_ON_DEMAND)
	req2 := makeRequest("r2", "c2", storage.ReportStatus_ON_DEMAND)
	req3 := makeRequest("r3", "c3", storage.ReportStatus_SCHEDULED)

	q.Enqueue(req1)
	q.Enqueue(req2)
	q.Enqueue(req3)
	assert.Equal(t, 3, q.Len())

	got := q.Dequeue()
	assert.Equal(t, "r1", got.ReportSnapshot.GetReportId())
	got = q.Dequeue()
	assert.Equal(t, "r2", got.ReportSnapshot.GetReportId())
	got = q.Dequeue()
	assert.Equal(t, "r3", got.ReportSnapshot.GetReportId())

	assert.Nil(t, q.Dequeue())
	assert.Equal(t, 0, q.Len())
}

func TestDequeueSkipsRunningConfig(t *testing.T) {
	q := New()

	req1 := makeRequest("r1", "c1", storage.ReportStatus_ON_DEMAND)
	req2 := makeRequest("r2", "c1", storage.ReportStatus_ON_DEMAND)
	req3 := makeRequest("r3", "c2", storage.ReportStatus_ON_DEMAND)

	q.Enqueue(req1)
	q.Enqueue(req2)
	q.Enqueue(req3)

	// Dequeue r1 → c1 is now in runningConfigs
	got := q.Dequeue()
	assert.Equal(t, "r1", got.ReportSnapshot.GetReportId())

	// r2 is for c1 (running), should be skipped; r3 (c2) is returned
	got = q.Dequeue()
	assert.Equal(t, "r3", got.ReportSnapshot.GetReportId())

	// r2 still blocked
	assert.Nil(t, q.Dequeue())
	assert.Equal(t, 1, q.Len())
}

func TestDequeueViewBasedAlwaysRunnable(t *testing.T) {
	q := New()

	req1 := makeRequest("r1", "c1", storage.ReportStatus_ON_DEMAND)
	reqView := makeRequest("r-view", "", storage.ReportStatus_VIEW_BASED)

	q.Enqueue(req1)
	q.Enqueue(reqView)

	// Dequeue r1 → c1 in runningConfigs
	got := q.Dequeue()
	assert.Equal(t, "r1", got.ReportSnapshot.GetReportId())

	// VIEW_BASED always passes regardless of runningConfigs
	got = q.Dequeue()
	require.NotNil(t, got)
	assert.Equal(t, "r-view", got.ReportSnapshot.GetReportId())
}

func TestMarkReportDoneForConfigUnblocks(t *testing.T) {
	q := New()

	req1 := makeRequest("r1", "c1", storage.ReportStatus_ON_DEMAND)
	req2 := makeRequest("r2", "c1", storage.ReportStatus_SCHEDULED)

	q.Enqueue(req1)
	q.Enqueue(req2)

	got := q.Dequeue()
	assert.Equal(t, "r1", got.ReportSnapshot.GetReportId())

	// r2 blocked because c1 is running
	assert.Nil(t, q.Dequeue())

	q.MarkReportDoneForConfig("c1")

	// Now r2 should be dequeued
	got = q.Dequeue()
	require.NotNil(t, got)
	assert.Equal(t, "r2", got.ReportSnapshot.GetReportId())
}

func TestRemoveFromQueue(t *testing.T) {
	q := New()

	req1 := makeRequest("r1", "c1", storage.ReportStatus_ON_DEMAND)
	req2 := makeRequest("r2", "c2", storage.ReportStatus_ON_DEMAND)

	q.Enqueue(req1)
	q.Enqueue(req2)

	removed := q.Remove("r1")
	require.NotNil(t, removed)
	assert.Equal(t, "r1", removed.ReportSnapshot.GetReportId())
	assert.Equal(t, 1, q.Len())

	// Removing non-existent returns nil
	assert.Nil(t, q.Remove("nonexistent"))
	assert.Equal(t, 1, q.Len())
}

func TestAddCancelFuncAndTryCancel(t *testing.T) {
	q := New()

	var capturedCause error
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	q.AddCancelFunc("r1", cancel)

	cancelled := q.TryCancel("r1")
	assert.True(t, cancelled)

	capturedCause = context.Cause(ctx)
	assert.ErrorIs(t, capturedCause, reportGen.ErrUserCancelled)
}

func TestTryCancelReturnsFalseForUnknown(t *testing.T) {
	q := New()
	assert.False(t, q.TryCancel("nonexistent"))
}

func TestRemoveCancelFunc(t *testing.T) {
	q := New()

	_, cancel := context.WithCancelCause(context.Background())

	q.AddCancelFunc("r1", cancel)
	q.RemoveCancelFunc("r1")

	// After removal, TryCancel should return false
	assert.False(t, q.TryCancel("r1"))
}

func TestViewBasedNotAddedToRunningConfigs(t *testing.T) {
	q := New()

	reqView1 := makeRequest("r-view-1", "", storage.ReportStatus_VIEW_BASED)
	reqView2 := makeRequest("r-view-2", "", storage.ReportStatus_VIEW_BASED)

	q.Enqueue(reqView1)
	q.Enqueue(reqView2)

	got := q.Dequeue()
	assert.Equal(t, "r-view-1", got.ReportSnapshot.GetReportId())

	// Second VIEW_BASED should also dequeue without being blocked
	got = q.Dequeue()
	require.NotNil(t, got)
	assert.Equal(t, "r-view-2", got.ReportSnapshot.GetReportId())
}
