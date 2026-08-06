package v2

import (
	"testing"
	"time"

	"github.com/stackrox/rox/central/reports/config/datastore/mocks"
	snapshotMocks "github.com/stackrox/rox/central/reports/snapshot/datastore/mocks"
	collectionMocks "github.com/stackrox/rox/central/resourcecollection/datastore/mocks"
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

func TestQueuePendingReports(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSnapshotStore := snapshotMocks.NewMockDataStore(ctrl)
	mockReportConfigDS := mocks.NewMockDataStore(ctrl)
	mockCollectionDS := collectionMocks.NewMockDataStore(ctrl)

	viewBasedSnap := &storage.ReportSnapshot{
		ReportId: "view-based-report-1",
		Name:     "View Based Report",
		Type:     storage.ReportSnapshot_VULNERABILITY,
		ReportStatus: &storage.ReportStatus{
			RunState:          storage.ReportStatus_PREPARING,
			ReportRequestType: storage.ReportStatus_VIEW_BASED,
		},
		Filter: &storage.ReportSnapshot_ViewBasedVulnReportFilters{
			ViewBasedVulnReportFilters: &storage.ViewBasedVulnerabilityReportFilters{
				Query: "CVE Type:IMAGE_CVE",
			},
		},
		Requester: &storage.SlimUser{Id: "user-1", Name: "user-1"},
	}

	configBasedSnap := &storage.ReportSnapshot{
		ReportId:              "config-based-report-1",
		ReportConfigurationId: "config-1",
		Name:                  "Config Based Report",
		Type:                  storage.ReportSnapshot_VULNERABILITY,
		ReportStatus: &storage.ReportStatus{
			RunState:          storage.ReportStatus_WAITING,
			ReportRequestType: storage.ReportStatus_ON_DEMAND,
		},
		Filter: &storage.ReportSnapshot_VulnReportFilters{
			VulnReportFilters: &storage.VulnerabilityReportFilters{},
		},
		ResourceScope: &storage.ResourceScope{
			ScopeReference: &storage.ResourceScope_CollectionId{CollectionId: "collection-1"},
		},
		Collection: &storage.CollectionSnapshot{Id: "collection-1", Name: "collection-1"},
		Requester:  &storage.SlimUser{Id: "user-2", Name: "user-2"},
	}

	mockSnapshotStore.EXPECT().
		SearchReportSnapshots(gomock.Any(), gomock.Any()).
		Return([]*storage.ReportSnapshot{viewBasedSnap, configBasedSnap}, nil)

	// View-based report should NOT trigger a config lookup.
	// Config-based report should trigger a config lookup.
	mockReportConfigDS.EXPECT().
		GetReportConfiguration(gomock.Any(), "config-1").
		Return(&storage.ReportConfiguration{Id: "config-1"}, true, nil)

	mockCollectionDS.EXPECT().
		Get(gomock.Any(), "collection-1").
		Return(&storage.ResourceCollection{Id: "collection-1"}, true, nil)

	// Both should be updated via resubmission.
	mockSnapshotStore.EXPECT().UpdateReportSnapshot(gomock.Any(), gomock.Any()).Return(nil).Times(2)

	cronScheduler := cron.New()
	cronScheduler.Start()
	defer cronScheduler.Stop()

	s := newSchedulerImpl(mockReportConfigDS, mockSnapshotStore, mockCollectionDS, nil, nil, nil, cronScheduler, nil)
	s.queuePendingReports()

	assert.Equal(t, 2, s.reportRequestsQueue.Len())
}
