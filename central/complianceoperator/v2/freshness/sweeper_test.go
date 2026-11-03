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

func TestExpectedRefresh(t *testing.T) {
	now := time.Now()

	t.Run("nil schedule returns false", func(t *testing.T) {
		_, ok := expectedRefresh(nil, now)
		assert.False(t, ok)
	})

	t.Run("valid daily schedule returns previous daily tick", func(t *testing.T) {
		sched := dailySchedule(3, 30)
		cron, err := schedule.ConvertToCronTab(sched)
		require.NoError(t, err)
		want, err := gronx.PrevTickBefore(cron, now, false)
		require.NoError(t, err)

		got, ok := expectedRefresh(sched, now)
		require.True(t, ok)
		assert.True(t, got.Equal(want), "got %s, want %s", got, want)
		assert.True(t, got.Before(now))
	})

	t.Run("unconvertible schedule returns false", func(t *testing.T) {
		// An UNSET interval does not convert to a cron expression.
		_, ok := expectedRefresh(&storage.Schedule{IntervalType: storage.Schedule_UNSET}, now)
		assert.False(t, ok)
	})
}

func TestSweep(t *testing.T) {
	grace := env.ComplianceScanScheduleWatcherTimeout.DurationSetting() + env.ComplianceScanWatcherTimeout.DurationSetting()

	t.Run("scheduled config is marked stale with cutoff = prevTick - grace", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		scanConfigMock := scanConfigMocks.NewMockDataStore(ctrl)
		checkResultMock := checkResultMocks.NewMockDataStore(ctrl)
		sw := newTestSweeper(scanConfigMock, checkResultMock)

		sched := dailySchedule(3, 30)
		cron, err := schedule.ConvertToCronTab(sched)
		require.NoError(t, err)

		scanConfigMock.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).
			Return([]*storage.ComplianceOperatorScanConfigurationV2{scanConfig("cfgA", sched)}, nil)

		var gotOlderThan time.Time
		checkResultMock.EXPECT().
			MarkScanChecksStaleByConfig(gomock.Any(), "cfgA", gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, olderThan time.Time) (int64, error) {
				gotOlderThan = olderThan
				return 1, nil
			})

		before := time.Now()
		sw.sweep()

		// The daily tick has minute granularity, so the previous tick computed from `before` matches
		// the one the sweeper computed from its own time.Now() a few microseconds later.
		prev, err := gronx.PrevTickBefore(cron, before, false)
		require.NoError(t, err)
		want := prev.Add(-grace)
		assert.True(t, gotOlderThan.Equal(want), "olderThan = %s, want prevTick(%s) - grace(%s) = %s", gotOlderThan, prev, grace, want)
	})

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
