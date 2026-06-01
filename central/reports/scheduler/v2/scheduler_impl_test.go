package v2

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/central/reports/config/datastore/mocks"
	reportGen "github.com/stackrox/rox/central/reports/scheduler/v2/reportgenerator"
	reportGenMocks "github.com/stackrox/rox/central/reports/scheduler/v2/reportgenerator/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"gopkg.in/robfig/cron.v2"
)

func TestFindPreviousFireTime(t *testing.T) {
	// Note: robfig/cron.v2 interprets cron specs in the system's local timezone.
	// In production, Central runs in UTC. Tests use time.Local to be portable.
	loc := time.Now().Location()

	var cases = []struct {
		testname     string
		cronSpec     string
		now          time.Time
		expectedTime time.Time
	}{
		{
			testname:     "Daily at 14:30, now is 16:00 same day",
			cronSpec:     "30 14 * * *",
			now:          time.Date(2026, 2, 16, 16, 0, 0, 0, loc),
			expectedTime: time.Date(2026, 2, 16, 14, 30, 0, 0, loc),
		},
		{
			testname:     "Daily at 14:30, now is 10:00 next day - missed yesterday's run",
			cronSpec:     "30 14 * * *",
			now:          time.Date(2026, 2, 17, 10, 0, 0, 0, loc),
			expectedTime: time.Date(2026, 2, 16, 14, 30, 0, 0, loc),
		},
		{
			testname:     "Weekly Monday at 09:00, now is Wednesday",
			cronSpec:     "0 9 * * 1",
			now:          time.Date(2026, 2, 18, 12, 0, 0, 0, loc), // Wednesday
			expectedTime: time.Date(2026, 2, 16, 9, 0, 0, 0, loc),  // Previous Monday
		},
		{
			testname:     "Daily at 00:00, now is 23:59 same day",
			cronSpec:     "0 0 * * *",
			now:          time.Date(2026, 2, 16, 23, 59, 0, 0, loc),
			expectedTime: time.Date(2026, 2, 16, 0, 0, 0, 0, loc),
		},
		{
			testname:     "Daily at 23:59, now is 00:01 next day",
			cronSpec:     "59 23 * * *",
			now:          time.Date(2026, 2, 17, 0, 1, 0, 0, loc),
			expectedTime: time.Date(2026, 2, 16, 23, 59, 0, 0, loc),
		},
	}

	for _, c := range cases {
		t.Run(c.testname, func(t *testing.T) {
			schedule, err := cron.Parse(c.cronSpec)
			assert.NoError(t, err)

			previousFire := findPreviousFireTime(schedule, c.now)
			assert.Equal(t, c.expectedTime, previousFire)
		})
	}
}

func TestFindPreviousFireTimeReturnsZeroWhenNoFireInWindow(t *testing.T) {
	// Monthly on the 15th at 10:00, now is Jan 16.
	// The lookback window is 32 days, starting from Dec 15.
	// Dec 15 at 10:00 is the only fire in that window, so it should be found.
	// But if the schedule fires only on Feb 29 (leap year), and now is Jan 1,
	// there may be no fire in the 32-day window. Use a far-future date to simulate.
	loc := time.Now().Location()
	schedule, err := cron.Parse("0 10 29 2 *") // Only Feb 29
	assert.NoError(t, err)

	// Now is March 1, 2027 (non-leap year). Feb 29 doesn't exist, so no fire in window.
	previousFire := findPreviousFireTime(schedule, time.Date(2027, 3, 1, 0, 0, 0, 0, loc))
	assert.True(t, previousFire.IsZero(), "Expected zero time when no fire exists in lookback window")
}

func TestQueueScheduledReportsSkipsEmptyResourceScope(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockReportConfigDS := mocks.NewMockDataStore(ctrl)

	weeklySchedule := &storage.Schedule{
		IntervalType: storage.Schedule_WEEKLY,
		Interval: &storage.Schedule_DaysOfWeek_{
			DaysOfWeek: &storage.Schedule_DaysOfWeek{
				Days: []int32{1},
			},
		},
	}

	validCollectionScope := &storage.ReportConfiguration{
		Id:   "config-with-collection",
		Name: "Valid Collection Config",
		Type: storage.ReportConfiguration_VULNERABILITY,
		ResourceScope: &storage.ResourceScope{
			ScopeReference: &storage.ResourceScope_CollectionId{CollectionId: "collection-1"},
		},
		Schedule: weeklySchedule,
	}
	validEntityScope := &storage.ReportConfiguration{
		Id:   "config-with-entity-scope",
		Name: "Valid Entity Scope Config",
		Type: storage.ReportConfiguration_VULNERABILITY,
		ResourceScope: &storage.ResourceScope{
			ScopeReference: &storage.ResourceScope_EntityScope{
				EntityScope: &storage.EntityScope{
					Rules: []*storage.EntityScopeRule{
						{
							Entity: storage.EntityType_ENTITY_TYPE_NAMESPACE,
							Field:  storage.EntityField_FIELD_NAME,
							Values: []*storage.RuleValue{{Value: "production"}},
						},
					},
				},
			},
		},
		Schedule: weeklySchedule,
	}
	emptyResourceScope := &storage.ReportConfiguration{
		Id:            "config-empty-scope",
		Name:          "Empty Scope Config (downgrade scenario)",
		Type:          storage.ReportConfiguration_VULNERABILITY,
		ResourceScope: &storage.ResourceScope{},
		Schedule:      weeklySchedule,
	}
	nilResourceScope := &storage.ReportConfiguration{
		Id:            "config-nil-scope",
		Name:          "Nil Scope Config",
		Type:          storage.ReportConfiguration_VULNERABILITY,
		ResourceScope: nil,
		Schedule:      weeklySchedule,
	}

	mockReportConfigDS.EXPECT().
		GetReportConfigurations(gomock.Any(), gomock.Any()).
		Return([]*storage.ReportConfiguration{
			validCollectionScope,
			validEntityScope,
			emptyResourceScope,
			nilResourceScope,
		}, nil)

	cronScheduler := cron.New()
	cronScheduler.Start()
	defer cronScheduler.Stop()

	s := newSchedulerImpl(mockReportConfigDS, nil, nil, nil, nil, nil, cronScheduler, nil)
	s.queueScheduledReports()

	// Only the two valid configs should have been scheduled
	assert.Len(t, s.reportConfigToEntryIDs, 2)
	assert.Contains(t, s.reportConfigToEntryIDs, "config-with-collection")
	assert.Contains(t, s.reportConfigToEntryIDs, "config-with-entity-scope")
	assert.NotContains(t, s.reportConfigToEntryIDs, "config-empty-scope")
	assert.NotContains(t, s.reportConfigToEntryIDs, "config-nil-scope")
}

func TestCancelRunningReportCancelsContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockReportGen := reportGenMocks.NewMockReportGenerator(ctrl)

	cronScheduler := cron.New()
	cronScheduler.Start()
	defer cronScheduler.Stop()

	s := newSchedulerImpl(nil, nil, nil, nil, mockReportGen, nil, cronScheduler, nil)

	started := make(chan struct{})
	done := make(chan struct{})
	var capturedCtx context.Context

	mockReportGen.EXPECT().ProcessReportRequest(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, req *reportGen.ReportRequest) {
			capturedCtx = ctx
			close(started)
			<-done
		},
	)

	req := &reportGen.ReportRequest{
		ReportSnapshot: &storage.ReportSnapshot{
			ReportId:              "test-report-id",
			ReportConfigurationId: "test-config-id",
			ReportStatus: &storage.ReportStatus{
				RunState: storage.ReportStatus_WAITING,
			},
		},
	}

	go s.runSingleReport(req)
	<-started

	cancelled := s.tryCancelRunningReport("test-report-id")
	assert.True(t, cancelled)

	assert.Error(t, capturedCtx.Err())
	assert.ErrorIs(t, context.Cause(capturedCtx), reportGen.ErrUserCancelled)

	close(done)
}

func TestCancelReportRequestCancelsRunningReport(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockReportGen := reportGenMocks.NewMockReportGenerator(ctrl)

	cronScheduler := cron.New()
	cronScheduler.Start()
	defer cronScheduler.Stop()

	s := newSchedulerImpl(nil, nil, nil, nil, mockReportGen, nil, cronScheduler, nil)

	started := make(chan struct{})
	done := make(chan struct{})
	var capturedCtx context.Context

	mockReportGen.EXPECT().ProcessReportRequest(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, req *reportGen.ReportRequest) {
			capturedCtx = ctx
			close(started)
			<-done
		},
	)

	req := &reportGen.ReportRequest{
		ReportSnapshot: &storage.ReportSnapshot{
			ReportId:              "running-report-id",
			ReportConfigurationId: "test-config-id",
			ReportStatus: &storage.ReportStatus{
				RunState: storage.ReportStatus_WAITING,
			},
		},
	}

	go s.runSingleReport(req)
	<-started

	// Report is not in queue (it's running), so CancelReportRequest should cancel the running context
	cancelled, err := s.CancelReportRequest(context.Background(), "running-report-id")
	assert.NoError(t, err)
	assert.True(t, cancelled)

	assert.Error(t, capturedCtx.Err())
	assert.ErrorIs(t, context.Cause(capturedCtx), reportGen.ErrUserCancelled)

	close(done)
}

func TestCancelReportRequestReturnsFalseForUnknownReport(t *testing.T) {
	cronScheduler := cron.New()
	cronScheduler.Start()
	defer cronScheduler.Stop()

	s := newSchedulerImpl(nil, nil, nil, nil, nil, nil, cronScheduler, nil)

	cancelled, err := s.CancelReportRequest(context.Background(), "nonexistent-id")
	assert.NoError(t, err)
	assert.False(t, cancelled)
}
