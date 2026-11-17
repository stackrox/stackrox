package freshness

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	checkResultMocks "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore/mocks"
	scanConfigMocks "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/testutils/goleak"
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

func weeklySchedule(day, hour, minute int32) *storage.Schedule {
	return &storage.Schedule{
		IntervalType: storage.Schedule_WEEKLY,
		Hour:         hour,
		Minute:       minute,
		Interval:     &storage.Schedule_Weekly{Weekly: &storage.Schedule_WeeklyInterval{Day: day}},
	}
}

func monthlySchedule(hour, minute int32, days ...int32) *storage.Schedule {
	return &storage.Schedule{
		IntervalType: storage.Schedule_MONTHLY,
		Hour:         hour,
		Minute:       minute,
		Interval:     &storage.Schedule_DaysOfMonth_{DaysOfMonth: &storage.Schedule_DaysOfMonth{Days: days}},
	}
}

func scanConfig(name string, sched *storage.Schedule) *storage.ComplianceOperatorScanConfigurationV2 {
	return &storage.ComplianceOperatorScanConfigurationV2{
		ScanConfigName: name,
		Schedule:       sched,
	}
}

// newTestSweeper returns a sweeper wired to fresh gomock datastores. The gomock controller is bound
// to t, so expectations are verified when the (sub)test ends.
func newTestSweeper(t *testing.T) (*sweeperImpl, *scanConfigMocks.MockDataStore, *checkResultMocks.MockDataStore) {
	ctrl := gomock.NewController(t)
	scanConfigMock := scanConfigMocks.NewMockDataStore(ctrl)
	checkResultMock := checkResultMocks.NewMockDataStore(ctrl)
	return &sweeperImpl{
		scanConfigDS:  scanConfigMock,
		checkResultDS: checkResultMock,
		stopper:       concurrency.NewStopper(),
	}, scanConfigMock, checkResultMock
}

// TestPreviousScheduledStart pins the cutoff computation against fixed references with hand-computed
// expected values (no gronx call in the test, so it cannot be circular). Schedules encode their
// hour/minute in UTC; the non-UTC-ref case guards against evaluating them in Central's local zone,
// which would shift the cutoff (the H1 timezone bug).
func TestPreviousScheduledStart(t *testing.T) {
	utc := func(y int, m time.Month, d, h, min int) time.Time {
		return time.Date(y, m, d, h, min, 0, 0, time.UTC)
	}
	newYork, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	cases := map[string]struct {
		sched *storage.Schedule
		ref   time.Time
		want  time.Time // zero value means ok==false is expected
	}{
		"nil schedule returns false": {
			sched: nil,
			ref:   utc(2026, time.January, 15, 10, 0),
		},
		"unconvertible schedule returns false": {
			// An UNSET interval does not convert to a cron expression.
			sched: &storage.Schedule{IntervalType: storage.Schedule_UNSET},
			ref:   utc(2026, time.January, 15, 10, 0),
		},
		"daily: previous tick is today": {
			sched: dailySchedule(3, 30),
			ref:   utc(2026, time.January, 15, 10, 0),
			want:  utc(2026, time.January, 15, 3, 30),
		},
		"daily: ref before today's tick crosses to yesterday": {
			sched: dailySchedule(3, 30),
			ref:   utc(2026, time.January, 15, 2, 0),
			want:  utc(2026, time.January, 14, 3, 30),
		},
		"daily: ref exactly on a tick returns the strictly-earlier tick": {
			sched: dailySchedule(3, 30),
			ref:   utc(2026, time.January, 15, 3, 30),
			want:  utc(2026, time.January, 14, 3, 30),
		},
		"daily: non-UTC ref is evaluated in UTC": {
			// 2026-01-15 05:00 New York == 2026-01-15 10:00 UTC, so the cutoff must match the UTC
			// mid-morning case. Evaluating the cron in the New York zone would instead yield
			// 2026-01-15 03:30 New York (08:30 UTC).
			sched: dailySchedule(3, 30),
			ref:   time.Date(2026, time.January, 15, 5, 0, 0, 0, newYork),
			want:  utc(2026, time.January, 15, 3, 30),
		},
		"weekly: previous tick is the scheduled weekday": {
			// 2026-01-15 is a Thursday (weekday 4); 03:30 precedes the 10:00 ref.
			sched: weeklySchedule(4, 3, 30),
			ref:   utc(2026, time.January, 15, 10, 0),
			want:  utc(2026, time.January, 15, 3, 30),
		},
		"monthly: previous tick is the scheduled day of month": {
			sched: monthlySchedule(3, 30, 10),
			ref:   utc(2026, time.January, 15, 10, 0),
			want:  utc(2026, time.January, 10, 3, 30),
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got, ok := previousScheduledStart(c.sched, c.ref)
			if c.want.IsZero() {
				assert.False(t, ok)
				return
			}
			require.True(t, ok)
			assert.Equal(t, c.want, got)
		})
	}
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

		// start is a scheduled scan start on a whole-minute boundary (so it coincides with a cron
		// tick) just after the bubble's fixed epoch, so "now" begins strictly before it. It is built
		// in UTC to match how the sweeper evaluates schedules; the previous cycle's results carry
		// LastStartedTime == prevCycle.
		start := time.Now().UTC().Truncate(time.Minute).Add(time.Minute)
		sched := dailySchedule(int32(start.Hour()), int32(start.Minute()))
		prevCycle := start.Add(-24 * time.Hour)

		sw, scanConfigMock, checkResultMock := newTestSweeper(t)

		scanConfigMock.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).
			Return([]*storage.ComplianceOperatorScanConfigurationV2{scanConfig("cfgA", sched)}, nil).AnyTimes()

		// assertCutoff sweeps at the bubble's current fake time and asserts the exact cutoff passed to
		// MarkScanChecksStaleByConfig. markStale uses a strict "<", so the previous cycle is marked
		// stale iff prevCycle predates the cutoff: prevCycle keeps it CURRENT, start flips it STALE.
		assertCutoff := func(wantCutoff time.Time) {
			t.Helper()
			var gotCutoff time.Time
			checkResultMock.EXPECT().
				MarkScanChecksStaleByConfig(gomock.Any(), "cfgA", gomock.Any()).
				DoAndReturn(func(_ context.Context, _ string, cutoff time.Time) (int64, error) {
					gotCutoff = cutoff
					return 0, nil
				})

			sw.sweep(context.Background())

			assert.Equal(t, wantCutoff, gotCutoff)
		}

		assertCutoff(prevCycle) // well before the scheduled start: previous cycle still current

		time.Sleep(time.Until(start.Add(time.Minute)))
		assertCutoff(prevCycle) // the new scan is running, within its grace window

		time.Sleep(time.Until(start.Add(grace)))
		assertCutoff(prevCycle) // exactly at the grace boundary: still current

		time.Sleep(time.Until(start.Add(grace + time.Minute)))
		assertCutoff(start) // grace fully elapsed with no new results: previous cycle now stale
	})
}

func TestSweep(t *testing.T) {
	t.Run("config with nil schedule is skipped", func(t *testing.T) {
		sw, scanConfigMock, _ := newTestSweeper(t)

		scanConfigMock.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).
			Return([]*storage.ComplianceOperatorScanConfigurationV2{scanConfig("cfgNil", nil)}, nil)
		// No MarkScanChecksStaleByConfig expectation: gomock fails if it is called.

		sw.sweep(context.Background())
	})

	t.Run("config updated after its previous tick is skipped", func(t *testing.T) {
		sw, scanConfigMock, _ := newTestSweeper(t)

		// A config just updated: its previous scheduled tick predates LastUpdatedTime, so the current
		// schedule was not in effect then and the config must be skipped this sweep.
		cfg := scanConfig("cfgRecentlyUpdated", dailySchedule(3, 30))
		cfg.LastUpdatedTime = protocompat.TimestampNow()
		scanConfigMock.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).
			Return([]*storage.ComplianceOperatorScanConfigurationV2{cfg}, nil)
		// No MarkScanChecksStaleByConfig expectation: gomock fails if it is called.

		sw.sweep(context.Background())
	})

	t.Run("list error is tolerated and marks nothing", func(t *testing.T) {
		sw, scanConfigMock, _ := newTestSweeper(t)

		scanConfigMock.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("boom"))
		// No MarkScanChecksStaleByConfig expectation.

		sw.sweep(context.Background())
	})

	t.Run("mark error on one config does not stop the next", func(t *testing.T) {
		sw, scanConfigMock, checkResultMock := newTestSweeper(t)

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

		sw.sweep(context.Background())
	})

	t.Run("stop requested between configs ends the loop early", func(t *testing.T) {
		sw, scanConfigMock, checkResultMock := newTestSweeper(t)

		sched := dailySchedule(3, 30)
		scanConfigMock.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).
			Return([]*storage.ComplianceOperatorScanConfigurationV2{
				scanConfig("cfg1", sched),
				scanConfig("cfg2", sched),
			}, nil)

		// Requesting a stop while processing cfg1 must make the loop skip cfg2: gomock fails the test
		// if MarkScanChecksStaleByConfig is called for cfg2, since no expectation is set for it.
		checkResultMock.EXPECT().MarkScanChecksStaleByConfig(gomock.Any(), "cfg1", gomock.Any()).
			DoAndReturn(func(context.Context, string, time.Time) (int64, error) {
				sw.stopper.Client().Stop()
				return 0, nil
			})

		sw.sweep(context.Background())
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
	defer goleak.AssertNoGoroutineLeaks(t)

	// The loop runs sweep immediately and then every tick; tolerate any number of list calls, and
	// no stale marking since there are no configurations.
	_, scanConfigMock, checkResultMock := newTestSweeper(t)
	scanConfigMock.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	s := New(scanConfigMock, checkResultMock, 10*time.Millisecond)
	s.Start()
	s.Stop()
}

// TestSweeperRunsInitialSweepBeforeFirstTick asserts loop() sweeps once immediately on Start(),
// rather than waiting for the first ticker fire. The interval is long enough that only the initial
// sweep could plausibly complete within the test's timeout.
func TestSweeperRunsInitialSweepBeforeFirstTick(t *testing.T) {
	_, scanConfigMock, checkResultMock := newTestSweeper(t)

	called := make(chan struct{}, 1)
	scanConfigMock.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, *v1.Query) ([]*storage.ComplianceOperatorScanConfigurationV2, error) {
			select {
			case called <- struct{}{}:
			default:
			}
			return nil, nil
		}).AnyTimes()

	s := New(scanConfigMock, checkResultMock, time.Hour)
	s.Start()
	defer s.Stop()

	select {
	case <-called:
	case <-time.After(5 * time.Second):
		t.Fatal("expected loop to run an initial sweep before the first tick")
	}
}
