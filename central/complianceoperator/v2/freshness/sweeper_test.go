package freshness

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/adhocore/gronx"
	checkResultMocks "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore/mocks"
	scanConfigMocks "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/protoconv/schedule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func dailySchedule(hour, minute int32) *storage.Schedule {
	return &storage.Schedule{
		IntervalType: storage.Schedule_DAILY,
		Hour:         hour,
		Minute:       minute,
	}
}

func scanConfig(name string, sched *storage.Schedule) *storage.ComplianceOperatorScanConfigurationV2 {
	return &storage.ComplianceOperatorScanConfigurationV2{
		ScanConfigName: name,
		Schedule:       sched,
	}
}

func newTestSweeper(scanCfg *scanConfigMocks.MockDataStore, checkRes *checkResultMocks.MockDataStore) *sweeperImpl {
	return &sweeperImpl{
		scanConfigDS:  scanCfg,
		checkResultDS: checkRes,
		stopper:       concurrency.NewStopper(),
	}
}

func TestPreviousScheduledStart(t *testing.T) {
	ref := time.Now()

	t.Run("nil schedule returns false", func(t *testing.T) {
		_, ok := previousScheduledStart(nil, ref)
		assert.False(t, ok)
	})

	t.Run("valid daily schedule returns previous daily tick", func(t *testing.T) {
		sched := dailySchedule(3, 30)
		cron, err := schedule.ConvertToCronTab(sched)
		require.NoError(t, err)
		want, err := gronx.PrevTickBefore(cron, ref, false)
		require.NoError(t, err)

		got, ok := previousScheduledStart(sched, ref)
		require.True(t, ok)
		assert.True(t, got.Equal(want), "got %s, want %s", got, want)
		assert.True(t, got.Before(ref))
	})

	t.Run("unconvertible schedule returns false", func(t *testing.T) {
		// An UNSET interval does not convert to a cron expression.
		_, ok := previousScheduledStart(&storage.Schedule{IntervalType: storage.Schedule_UNSET}, ref)
		assert.False(t, ok)
	})
}

// TestSweepGraceWindow pins the core staleness rule: a result is stale only once a scheduled start
// has passed AND its full grace window elapsed without new results. In particular, a fresh scheduled
// scan must not flip the previous cycle's results to STALE while that scan is still within its grace
// window (the bug yuumasato caught on PR #23851).
func TestSweepGraceWindow(t *testing.T) {
	grace := env.ComplianceScanScheduleWatcherTimeout.DurationSetting() + env.ComplianceScanWatcherTimeout.DurationSetting()
	require.Positive(t, grace)
	require.Less(t, grace, 24*time.Hour, "test assumes grace is shorter than the daily interval")

	sched := dailySchedule(3, 0)
	// start is a scheduled scan start; the previous cycle's results carry laststartedtime == prevCycle.
	start := time.Date(2026, 1, 2, 3, 0, 0, 0, time.UTC)
	prevCycle := start.Add(-24 * time.Hour)

	cases := map[string]struct {
		now        time.Time
		wantCutoff time.Time
	}{
		"within grace window keeps previous cycle current": {
			// The scan at start is running; its grace has not elapsed. The cutoff must not pass the
			// previous cycle's start, so markStale (strict <) leaves those results CURRENT.
			now:        start.Add(time.Minute),
			wantCutoff: prevCycle,
		},
		"at grace boundary still keeps previous cycle current": {
			now:        start.Add(grace),
			wantCutoff: prevCycle,
		},
		"after grace elapses marks the previous cycle stale": {
			// start's grace fully elapsed with no new results, so the cutoff advances to start and the
			// previous cycle (laststartedtime < start) becomes stale.
			now:        start.Add(grace + time.Minute),
			wantCutoff: start,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			scanConfigMock := scanConfigMocks.NewMockDataStore(ctrl)
			checkResultMock := checkResultMocks.NewMockDataStore(ctrl)
			sw := newTestSweeper(scanConfigMock, checkResultMock)

			scanConfigMock.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).
				Return([]*storage.ComplianceOperatorScanConfigurationV2{scanConfig("cfgA", sched)}, nil)

			var gotCutoff time.Time
			checkResultMock.EXPECT().
				MarkScanChecksStaleByConfig(gomock.Any(), "cfgA", gomock.Any()).
				DoAndReturn(func(_ context.Context, _ string, cutoff time.Time) (int64, error) {
					gotCutoff = cutoff
					return 0, nil
				})

			sw.sweep(tc.now)

			assert.True(t, gotCutoff.Equal(tc.wantCutoff), "cutoff = %s, want %s", gotCutoff, tc.wantCutoff)
			// markStale uses a strict "<", so the previous cycle is stale iff its start predates the cutoff.
			prevCycleStale := prevCycle.Before(gotCutoff)
			wantStale := tc.wantCutoff.Equal(start)
			assert.Equal(t, wantStale, prevCycleStale, "previous-cycle staleness mismatch for cutoff %s", gotCutoff)
		})
	}
}

func TestSweep(t *testing.T) {
	now := time.Now()

	t.Run("config with nil schedule is skipped", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		scanConfigMock := scanConfigMocks.NewMockDataStore(ctrl)
		checkResultMock := checkResultMocks.NewMockDataStore(ctrl)
		sw := newTestSweeper(scanConfigMock, checkResultMock)

		scanConfigMock.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).
			Return([]*storage.ComplianceOperatorScanConfigurationV2{scanConfig("cfgNil", nil)}, nil)
		// No MarkScanChecksStaleByConfig expectation: gomock fails if it is called.

		sw.sweep(now)
	})

	t.Run("list error is tolerated and marks nothing", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		scanConfigMock := scanConfigMocks.NewMockDataStore(ctrl)
		checkResultMock := checkResultMocks.NewMockDataStore(ctrl)
		sw := newTestSweeper(scanConfigMock, checkResultMock)

		scanConfigMock.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("boom"))
		// No MarkScanChecksStaleByConfig expectation.

		sw.sweep(now)
	})

	t.Run("mark error on one config does not stop the next", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		scanConfigMock := scanConfigMocks.NewMockDataStore(ctrl)
		checkResultMock := checkResultMocks.NewMockDataStore(ctrl)
		sw := newTestSweeper(scanConfigMock, checkResultMock)

		sched := dailySchedule(3, 30)
		scanConfigMock.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).
			Return([]*storage.ComplianceOperatorScanConfigurationV2{
				scanConfig("cfg1", sched),
				scanConfig("cfg2", sched),
			}, nil)

		checkResultMock.EXPECT().MarkScanChecksStaleByConfig(gomock.Any(), "cfg1", gomock.Any()).
			Return(int64(0), errors.New("boom"))
		checkResultMock.EXPECT().MarkScanChecksStaleByConfig(gomock.Any(), "cfg2", gomock.Any()).
			Return(int64(0), nil)

		sw.sweep(now)
	})
}

func TestSweeperStartStop(t *testing.T) {
	ctrl := gomock.NewController(t)
	scanConfigMock := scanConfigMocks.NewMockDataStore(ctrl)
	checkResultMock := checkResultMocks.NewMockDataStore(ctrl)

	// The loop runs sweep immediately and then every tick; tolerate any number of list calls, and
	// no stale marking since there are no configurations.
	scanConfigMock.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	s := New(scanConfigMock, checkResultMock, 10*time.Millisecond)
	s.Start()
	s.Stop()
}
