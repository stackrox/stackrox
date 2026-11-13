package freshness

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
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
//
// It runs inside a synctest bubble so sweep's internal time.Now() calls see a fake, controllable
// clock: time.Sleep advances the bubble's clock instantly instead of waiting on the wall clock, which
// lets the test walk up to and across the grace boundary deterministically without sweep needing a
// test-only "now" parameter.
func TestSweepGraceWindow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		grace := env.ComplianceScanScheduleWatcherTimeout.DurationSetting() + env.ComplianceScanWatcherTimeout.DurationSetting()
		require.Positive(t, grace)
		require.Less(t, grace, 24*time.Hour, "test assumes grace is shorter than the daily interval")

		// start is a scheduled scan start, anchored shortly after the bubble's fixed epoch so "now"
		// begins strictly before it. The previous cycle's results carry LastStartedTime == prevCycle.
		start := time.Now().Add(time.Minute)
		sched := dailySchedule(int32(start.Hour()), int32(start.Minute()))
		prevCycle := start.Add(-24 * time.Hour)

		ctrl := gomock.NewController(t)
		scanConfigMock := scanConfigMocks.NewMockDataStore(ctrl)
		checkResultMock := checkResultMocks.NewMockDataStore(ctrl)
		sw := newTestSweeper(scanConfigMock, checkResultMock)

		scanConfigMock.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).
			Return([]*storage.ComplianceOperatorScanConfigurationV2{scanConfig("cfgA", sched)}, nil).AnyTimes()

		// assertPrevCycleStale sweeps at the bubble's current fake time and checks whether the previous
		// cycle's results would be marked stale, via the cutoff passed to MarkScanChecksStaleByConfig.
		assertPrevCycleStale := func(wantStale bool) {
			t.Helper()
			var gotCutoff time.Time
			checkResultMock.EXPECT().
				MarkScanChecksStaleByConfig(gomock.Any(), "cfgA", gomock.Any()).
				DoAndReturn(func(_ context.Context, _ string, cutoff time.Time) (int64, error) {
					gotCutoff = cutoff
					return 0, nil
				})

			sw.sweep()

			// markStale uses a strict "<", so the previous cycle is stale iff its start predates cutoff.
			assert.Equal(t, wantStale, prevCycle.Before(gotCutoff), "cutoff = %s", gotCutoff)
		}

		assertPrevCycleStale(false) // well before the scheduled start

		time.Sleep(time.Until(start.Add(time.Minute)))
		assertPrevCycleStale(false) // the new scan is running, within its grace window

		time.Sleep(time.Until(start.Add(grace)))
		assertPrevCycleStale(false) // exactly at the grace boundary: still current

		time.Sleep(time.Until(start.Add(grace + time.Minute)))
		assertPrevCycleStale(true) // grace fully elapsed with no new results
	})
}

func TestSweep(t *testing.T) {
	t.Run("config with nil schedule is skipped", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		scanConfigMock := scanConfigMocks.NewMockDataStore(ctrl)
		checkResultMock := checkResultMocks.NewMockDataStore(ctrl)
		sw := newTestSweeper(scanConfigMock, checkResultMock)

		scanConfigMock.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).
			Return([]*storage.ComplianceOperatorScanConfigurationV2{scanConfig("cfgNil", nil)}, nil)
		// No MarkScanChecksStaleByConfig expectation: gomock fails if it is called.

		sw.sweep()
	})

	t.Run("list error is tolerated and marks nothing", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		scanConfigMock := scanConfigMocks.NewMockDataStore(ctrl)
		checkResultMock := checkResultMocks.NewMockDataStore(ctrl)
		sw := newTestSweeper(scanConfigMock, checkResultMock)

		scanConfigMock.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("boom"))
		// No MarkScanChecksStaleByConfig expectation.

		sw.sweep()
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

		sw.sweep()
	})
}

func TestClampSweepInterval(t *testing.T) {
	cases := map[string]struct {
		interval time.Duration
		want     time.Duration
	}{
		"below the minimum is clamped up": {
			interval: 10 * time.Second,
			want:     minSweepInterval,
		},
		"exactly the minimum is unchanged": {
			interval: minSweepInterval,
			want:     minSweepInterval,
		},
		"above the minimum is unchanged": {
			interval: 30 * time.Minute,
			want:     30 * time.Minute,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, c.want, clampSweepInterval(c.interval))
		})
	}
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
