package scheduler

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stackrox/rox/central/complianceoperator/v2/report"
	snapshotMocks "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore/mocks"
	generatorMocks "github.com/stackrox/rox/central/complianceoperator/v2/report/manager/generator/mocks"
	scanConfigMocks "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore/mocks"
	scanDSMocks "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore/mocks"
	bindingsMocks "github.com/stackrox/rox/central/complianceoperator/v2/scansettingbindings/datastore/mocks"
	suiteMocks "github.com/stackrox/rox/central/complianceoperator/v2/suites/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
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

func scanConfig(id, name string, sched *storage.Schedule) *storage.ComplianceOperatorScanConfigurationV2 {
	return &storage.ComplianceOperatorScanConfigurationV2{
		Id:             id,
		ScanConfigName: name,
		Schedule:       sched,
	}
}

// schedulerMocks bundles the scheduler's gomock dependencies.
type schedulerMocks struct {
	scanConfigDS *scanConfigMocks.MockDataStore
	scanDS       *scanDSMocks.MockDataStore
	snapshotDS   *snapshotMocks.MockDataStore
	suiteDS      *suiteMocks.MockDataStore
	bindingsDS   *bindingsMocks.MockDataStore
	reportGen    *generatorMocks.MockComplianceReportGenerator
}

// newTestScheduler returns a scheduler wired to fresh gomock datastores with the decoupled
// reporting feature enabled. The gomock controller is bound to t, so expectations are verified
// when the (sub)test ends.
func newTestScheduler(t *testing.T) (*managerImpl, *schedulerMocks) {
	t.Setenv(features.ComplianceReporting.EnvVar(), "true")
	t.Setenv(features.ComplianceDecoupledReports.EnvVar(), "true")
	ctrl := gomock.NewController(t)
	mocks := &schedulerMocks{
		scanConfigDS: scanConfigMocks.NewMockDataStore(ctrl),
		scanDS:       scanDSMocks.NewMockDataStore(ctrl),
		snapshotDS:   snapshotMocks.NewMockDataStore(ctrl),
		suiteDS:      suiteMocks.NewMockDataStore(ctrl),
		bindingsDS:   bindingsMocks.NewMockDataStore(ctrl),
		reportGen:    generatorMocks.NewMockComplianceReportGenerator(ctrl),
	}
	return New(mocks.scanConfigDS, mocks.scanDS, mocks.snapshotDS, mocks.suiteDS, mocks.bindingsDS, mocks.reportGen).(*managerImpl), mocks
}

// reportScheduleFor returns the report schedule of the manager's cron entry for the given scan
// configuration, or nil if it has none. All entry mutations in these tests happen on the test
// goroutine, so reading entryIDs without the lock is safe.
func reportScheduleFor(m *managerImpl, scanConfigID string) *reportSchedule {
	entryID, ok := m.entryIDs[scanConfigID]
	if !ok {
		return nil
	}
	for _, e := range m.cron.Entries() {
		if e.ID == entryID {
			s, _ := e.Schedule.(*reportSchedule)
			return s
		}
	}
	return nil
}

func TestNewReportSchedule(t *testing.T) {
	cases := map[string]struct {
		cfg      *storage.ComplianceOperatorScanConfigurationV2
		wantErr  bool
		wantCron string
	}{
		"daily schedule": {
			cfg:      scanConfig("cfg-daily", "cfg-daily", dailySchedule(1, 30)),
			wantCron: "30 1 * * *",
		},
		"weekly schedule": {
			cfg:      scanConfig("cfg-weekly", "cfg-weekly", weeklySchedule(5, 2, 0)),
			wantCron: "0 2 * * 5",
		},
		"one-time scan has no report schedule": {
			cfg: &storage.ComplianceOperatorScanConfigurationV2{
				Id:             "cfg-one-time",
				ScanConfigName: "cfg-one-time",
				OneTimeScan:    true,
				Schedule:       dailySchedule(1, 30),
			},
			wantErr: true,
		},
		"nil schedule": {
			cfg:     scanConfig("cfg-nil", "cfg-nil", nil),
			wantErr: true,
		},
		"unset interval": {
			cfg:     scanConfig("cfg-unset", "cfg-unset", &storage.Schedule{IntervalType: storage.Schedule_UNSET}),
			wantErr: true,
		},
		"missing id": {
			cfg:     scanConfig("", "cfg-noid", dailySchedule(1, 30)),
			wantErr: true,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			s, err := newReportSchedule(c.cfg)
			if c.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.wantCron, s.cronTab)
			assert.Equal(t, c.cfg.GetId(), s.scanConfigID)
			assert.Equal(t, c.cfg.GetScanConfigName(), s.scanConfigName)
		})
	}
}

// TestNextFireAfter pins the fire-time computation against fixed references with hand-computed
// expected values (no gronx call in the test setup, so it cannot be circular). Schedules encode
// their hour/minute in UTC; the non-UTC-ref case guards against evaluating them in Central's
// local zone, which would shift the fire time (the H1 timezone bug).
func TestNextFireAfter(t *testing.T) {
	// November 25 2026 is a Wednesday.
	utc := func(y int, m time.Month, d, h, min int) time.Time {
		return time.Date(y, m, d, h, min, 0, 0, time.UTC)
	}
	newYork, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	cases := map[string]struct {
		schedule *storage.Schedule
		delay    string
		ref      time.Time
		want     time.Time
	}{
		"cycle in progress still gets its report": {
			// At 00:30 the cycle that started at 00:00 has not been reported yet: its report is
			// due at 00:00+delay.
			schedule: dailySchedule(0, 0),
			delay:    "1h",
			ref:      utc(2026, time.November, 25, 0, 30),
			want:     utc(2026, time.November, 25, 1, 0),
		},
		"cycle whose report time passed is skipped": {
			// At 01:30 the 01:00 report time has passed; the next one is tomorrow's cycle.
			schedule: dailySchedule(0, 0),
			delay:    "1h",
			ref:      utc(2026, time.November, 25, 1, 30),
			want:     utc(2026, time.November, 26, 1, 0),
		},
		"ref exactly at a fire time schedules the next cycle": {
			schedule: dailySchedule(0, 0),
			delay:    "1h",
			ref:      utc(2026, time.November, 25, 1, 0),
			want:     utc(2026, time.November, 26, 1, 0),
		},
		"shorter delay pulls the report closer to the tick": {
			schedule: dailySchedule(0, 0),
			delay:    "30m",
			ref:      utc(2026, time.November, 25, 0, 15),
			want:     utc(2026, time.November, 25, 0, 30),
		},
		"weekly schedule fires on the scheduled weekday": {
			// The next Friday after the Wednesday reference is November 27.
			schedule: weeklySchedule(5, 2, 0),
			delay:    "1h",
			ref:      utc(2026, time.November, 25, 18, 0),
			want:     utc(2026, time.November, 27, 3, 0),
		},
		"monthly schedule fires on the scheduled day": {
			schedule: monthlySchedule(4, 0, 15),
			delay:    "2h",
			ref:      utc(2026, time.November, 25, 12, 0),
			want:     utc(2026, time.December, 15, 6, 0),
		},
		"non-UTC ref is evaluated in UTC": {
			// 00:00 New York on November 25 is 05:00 UTC, so the search starts at 04:00 UTC and
			// the next tick is November 26 03:00 UTC. Evaluating the cron in the New York zone
			// would instead yield November 25 04:00 New York (09:00 UTC).
			schedule: dailySchedule(3, 0),
			delay:    "1h",
			ref:      time.Date(2026, time.November, 25, 0, 0, 0, 0, newYork),
			want:     utc(2026, time.November, 26, 4, 0),
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv(env.ComplianceReportScheduleDelay.EnvVar(), c.delay)
			s, err := newReportSchedule(scanConfig("cfg", "cfg", c.schedule))
			require.NoError(t, err)

			got, err := s.NextFireAfter(c.ref)
			require.NoError(t, err)
			assert.Equal(t, c.want, got)
		})
	}
}

func TestJobQueue(t *testing.T) {
	q := newJobQueue()

	assert.Nil(t, q.dequeue(), "an empty queue has nothing to run")

	q.enqueue(&reportJob{scanConfigID: "cfg1", scanConfigName: "cfg-one"})
	q.enqueue(&reportJob{scanConfigID: "cfg2", scanConfigName: "cfg-two"})
	q.enqueue(&reportJob{scanConfigID: "cfg1", scanConfigName: "cfg-one"})

	// Jobs of different configurations run one at a time; a second job of a running
	// configuration waits.
	job := q.dequeue()
	require.NotNil(t, job)
	assert.Equal(t, "cfg1", job.scanConfigID)

	job = q.dequeue()
	require.NotNil(t, job)
	assert.Equal(t, "cfg2", job.scanConfigID)

	assert.Nil(t, q.dequeue(), "cfg1 is still running, so nothing is runnable")

	// Once the running job finishes, the queued job of the same configuration is runnable
	// again.
	q.markDone("cfg1")
	job = q.dequeue()
	require.NotNil(t, job)
	assert.Equal(t, "cfg1", job.scanConfigID)
}

// TestRunJobsRunsSubmittedJobs exercises the execution loop: jobs run asynchronously, a second
// job of a running configuration waits for it to finish, and jobs of other configurations are
// not blocked. Execution is observed at the report job's first datastore read, which the mock
// parks until released: a started job holds its configuration in the running set.
func TestRunJobsRunsSubmittedJobs(t *testing.T) {
	m, mocks := newTestScheduler(t)
	mocks.scanConfigDS.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	started := make(chan string, 4)
	release := make(chan struct{})
	mocks.scanConfigDS.EXPECT().GetScanConfiguration(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, scanConfigID string) (*storage.ComplianceOperatorScanConfigurationV2, bool, error) {
			started <- scanConfigID
			<-release
			// Not found: the job finishes without generating anything.
			return nil, false, nil
		}).AnyTimes()

	m.start()
	defer m.Stop()

	// The first job of cfg1 runs; a second submission of cfg1 must wait behind it.
	m.submitJob("cfg1", "cfg-one")
	m.submitJob("cfg1", "cfg-one")

	select {
	case id := <-started:
		assert.Equal(t, "cfg1", id)
	case <-time.After(5 * time.Second):
		t.Fatal("the first job did not run")
	}

	select {
	case <-started:
		t.Fatal("a second job of the same configuration ran concurrently with the first")
	case <-time.After(100 * time.Millisecond):
	}

	// A job of another configuration is not blocked by the running one.
	m.submitJob("cfg2", "cfg-two")
	select {
	case id := <-started:
		assert.Equal(t, "cfg2", id)
	case <-time.After(5 * time.Second):
		t.Fatal("a job of another configuration did not run")
	}

	// Releasing the running jobs lets the queued job of cfg1 run.
	close(release)
	select {
	case id := <-started:
		assert.Equal(t, "cfg1", id)
	case <-time.After(5 * time.Second):
		t.Fatal("the queued job did not run after the running one finished")
	}
}

func TestUpsertAndRemoveScanConfig(t *testing.T) {
	m, _ := newTestScheduler(t)

	m.UpsertScanConfig(scanConfig("cfg1", "cfg-daily", dailySchedule(0, 0)))
	require.Len(t, m.entryIDs, 1)
	require.Contains(t, m.entryIDs, "cfg1")
	h := reportScheduleFor(m, "cfg1")
	require.NotNil(t, h)
	assert.Equal(t, "0 0 * * *", h.cronTab)

	// Upserting the same configuration with a new schedule replaces the entry.
	m.UpsertScanConfig(scanConfig("cfg1", "cfg-daily", dailySchedule(12, 0)))
	require.Len(t, m.entryIDs, 1)
	h = reportScheduleFor(m, "cfg1")
	require.NotNil(t, h)
	assert.Equal(t, "0 12 * * *", h.cronTab)

	// Upserting a configuration that lost its evaluable schedule removes its entry: it is
	// reported on demand only.
	m.UpsertScanConfig(&storage.ComplianceOperatorScanConfigurationV2{
		Id:             "cfg1",
		ScanConfigName: "cfg-one-time",
		OneTimeScan:    true,
	})
	assert.Empty(t, m.entryIDs)

	// Upserting an unscheduled configuration when no entry exists is a no-op.
	m.UpsertScanConfig(scanConfig("cfg2", "cfg-nil", nil))
	assert.Empty(t, m.entryIDs)

	m.UpsertScanConfig(scanConfig("cfg3", "cfg-weekly", weeklySchedule(5, 2, 0)))
	assert.Len(t, m.entryIDs, 1)

	// Removing an unknown configuration is a no-op.
	m.RemoveScanConfig("unknown")
	assert.Len(t, m.entryIDs, 1)

	m.RemoveScanConfig("cfg3")
	assert.Empty(t, m.entryIDs)
}

func TestReconcile(t *testing.T) {
	t.Run("creates entries for configurations with evaluable schedules only", func(t *testing.T) {
		m, mocks := newTestScheduler(t)
		mocks.scanConfigDS.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).
			Return([]*storage.ComplianceOperatorScanConfigurationV2{
				scanConfig("cfg-daily", "cfg-daily", dailySchedule(1, 30)),
				scanConfig("cfg-nil", "cfg-nil", nil),
				{Id: "cfg-one-time", ScanConfigName: "cfg-one-time", OneTimeScan: true},
			}, nil)

		m.reconcile(context.Background())

		require.Len(t, m.entryIDs, 1)
		require.Contains(t, m.entryIDs, "cfg-daily")
		assert.NotNil(t, reportScheduleFor(m, "cfg-daily"))
	})

	t.Run("drops entries of configurations no longer stored", func(t *testing.T) {
		m, mocks := newTestScheduler(t)
		m.UpsertScanConfig(scanConfig("cfg-gone", "cfg-gone", dailySchedule(1, 30)))

		mocks.scanConfigDS.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).
			Return([]*storage.ComplianceOperatorScanConfigurationV2{
				scanConfig("cfg-kept", "cfg-kept", dailySchedule(1, 30)),
			}, nil)

		m.reconcile(context.Background())

		assert.NotContains(t, m.entryIDs, "cfg-gone")
		assert.Contains(t, m.entryIDs, "cfg-kept")
	})

	t.Run("list error keeps the existing entries", func(t *testing.T) {
		m, mocks := newTestScheduler(t)
		m.UpsertScanConfig(scanConfig("cfg-keep", "cfg-keep", dailySchedule(1, 30)))

		mocks.scanConfigDS.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("boom"))

		m.reconcile(context.Background())

		assert.Contains(t, m.entryIDs, "cfg-keep")
	})
}

func TestSchedulerStartStop(t *testing.T) {
	defer goleak.AssertNoGoroutineLeaks(t)

	// The loop reconciles immediately on Start; tolerate any number of list calls, and no
	// entries since there are no configurations.
	m, mocks := newTestScheduler(t)
	mocks.scanConfigDS.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	m.start()
	m.Stop()
}

// TestSchedulerRunsInitialReconcileOnStart asserts Start() reconciles the stored scan
// configurations synchronously, so configurations that predate the process get cron entries
// without producing events.
func TestSchedulerRunsInitialReconcileOnStart(t *testing.T) {
	m, mocks := newTestScheduler(t)

	called := make(chan struct{}, 1)
	mocks.scanConfigDS.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, *v1.Query) ([]*storage.ComplianceOperatorScanConfigurationV2, error) {
			select {
			case called <- struct{}{}:
			default:
			}
			return nil, nil
		}).AnyTimes()

	m.start()
	defer m.Stop()

	select {
	case <-called:
	case <-time.After(5 * time.Second):
		t.Fatal("expected the scheduler to reconcile on Start")
	}
}

// TestUpsertReordersUpcomingFires covers the update case of the ordering question: the cron
// loop sleeps until the soonest entry, but every add or remove wakes it and it re-sorts, so
// updating a configuration whose report was due later can move it ahead of another
// configuration's pending report.
func TestUpsertReordersUpcomingFires(t *testing.T) {
	t.Setenv(env.ComplianceReportScheduleDelay.EnvVar(), "1h")

	synctest.Test(t, func(t *testing.T) {
		m, mocks := newTestScheduler(t)

		start := time.Now().UTC().Truncate(time.Minute).Add(time.Minute)
		at := func(offset time.Duration) *storage.Schedule {
			tick := start.Add(offset)
			return dailySchedule(int32(tick.Hour()), int32(tick.Minute()))
		}

		// cfg1's report is due at start+110m and cfg2's at start+180m, so the pending order is
		// 1 then 2.
		mocks.scanConfigDS.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).
			Return([]*storage.ComplianceOperatorScanConfigurationV2{
				scanConfig("cfg1", "cfg-one", at(50*time.Minute)),
				scanConfig("cfg2", "cfg-two", at(120*time.Minute)),
			}, nil).AnyTimes()

		// A fired report job starts by reading its configuration; observe it there and let it
		// finish as "not found".
		fired := make(chan string, 4)
		mocks.scanConfigDS.EXPECT().GetScanConfiguration(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, scanConfigID string) (*storage.ComplianceOperatorScanConfigurationV2, bool, error) {
				select {
				case fired <- scanConfigID:
				default:
				}
				return nil, false, nil
			}).AnyTimes()

		m.start()
		defer m.Stop()

		drain := func() []string {
			var ids []string
			for {
				select {
				case id := <-fired:
					ids = append(ids, id)
				default:
					return ids
				}
			}
		}

		// Nothing fires before any report is due.
		time.Sleep(time.Until(start.Add(5 * time.Minute)))
		assert.Empty(t, drain())

		// Update cfg2 to a schedule whose report is due at start+80m, ahead of cfg1's
		// start+110m: the pending order flips to 2 then 1.
		m.UpsertScanConfig(scanConfig("cfg2", "cfg-two", at(20*time.Minute)))

		time.Sleep(time.Until(start.Add(85 * time.Minute)))
		assert.Equal(t, []string{"cfg2"}, drain())

		time.Sleep(time.Until(start.Add(115 * time.Minute)))
		assert.Equal(t, []string{"cfg1"}, drain())

		// The replaced entry is gone: cfg2 does not fire again at its old due time.
		time.Sleep(time.Until(start.Add(185 * time.Minute)))
		assert.Empty(t, drain())
	})
}

// TestSchedulerFiresOnSchedule runs the full loop inside a synctest bubble: with a daily
// schedule and a one-hour delay, the job must run once at exactly tick+delay and not before.
// The shared cron sleeps until the soonest entry's fire time, so there is no check interval
// adding latency.
func TestSchedulerFiresOnSchedule(t *testing.T) {
	t.Setenv(env.ComplianceReportScheduleDelay.EnvVar(), "1h")

	synctest.Test(t, func(t *testing.T) {
		m, mocks := newTestScheduler(t)

		// The schedule ticks one minute after the bubble epoch on a whole-minute boundary (so
		// it coincides with a cron tick), so "now" starts strictly before it; the report is due
		// an hour later.
		start := time.Now().UTC().Truncate(time.Minute).Add(time.Minute)
		sched := dailySchedule(int32(start.Hour()), int32(start.Minute()))

		mocks.scanConfigDS.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).
			Return([]*storage.ComplianceOperatorScanConfigurationV2{scanConfig("cfg1", "cfg-daily", sched)}, nil).AnyTimes()

		// A fired report job starts by reading its configuration; observe the fake clock there
		// and let the job finish as "not found".
		firedAt := make(chan time.Time, 4)
		mocks.scanConfigDS.EXPECT().GetScanConfiguration(gomock.Any(), "cfg1").
			DoAndReturn(func(context.Context, string) (*storage.ComplianceOperatorScanConfigurationV2, bool, error) {
				select {
				case firedAt <- time.Now():
				default:
				}
				return nil, false, nil
			}).AnyTimes()

		m.start()
		defer m.Stop()

		// Half an hour past the tick nothing has fired yet.
		select {
		case at := <-firedAt:
			t.Fatalf("report fired before tick+delay, at %v", at)
		case <-time.After(time.Until(start.Add(30 * time.Minute))):
		}

		// At tick+delay the report fires exactly.
		var at time.Time
		select {
		case at = <-firedAt:
		case <-time.After(time.Until(start.Add(time.Hour + time.Minute))):
			t.Fatal("report did not fire at tick+delay")
		}
		assert.False(t, at.Before(start.Add(time.Hour)), "report fired early: %v", at)
		assert.LessOrEqual(t, at.Sub(start.Add(time.Hour)), time.Second, "report fired late: %v", at)

		// The job does not refire for the same cycle.
		select {
		case at := <-firedAt:
			t.Fatalf("report refired for the same cycle, at %v", at)
		case <-time.After(time.Until(start.Add(90 * time.Minute))):
		}
	})
}

// TestStartDisabled asserts Start is a no-op while the rollout feature is disabled: nothing
// starts and no advisory lock is taken (the nil DB would panic otherwise).
func TestStartDisabled(t *testing.T) {
	m, _ := newTestScheduler(t)
	t.Setenv(features.ComplianceDecoupledReports.EnvVar(), "false")

	m.Start(nil)
	assert.False(t, m.isStarted.Load())
}

func TestScheduleRescanReport(t *testing.T) {
	t.Setenv(env.ComplianceReportScheduleDelay.EnvVar(), "1h")

	t.Run("unknown configuration is skipped", func(t *testing.T) {
		m, mocks := newTestScheduler(t)
		mocks.scanConfigDS.EXPECT().GetScanConfiguration(gomock.Any(), "cfg-unknown").
			Return(nil, false, nil)

		m.ScheduleRescanReport("cfg-unknown")
		assert.Empty(t, m.rescanEntryIDs)
	})

	t.Run("one-time scans get a re-run report", func(t *testing.T) {
		m, mocks := newTestScheduler(t)
		cfg := &storage.ComplianceOperatorScanConfigurationV2{Id: "cfg-ot", ScanConfigName: "cfg-one-time", OneTimeScan: true}
		mocks.scanConfigDS.EXPECT().GetScanConfiguration(gomock.Any(), "cfg-ot").
			Return(cfg, true, nil)

		m.ScheduleRescanReport("cfg-ot")
		assert.Len(t, m.rescanEntryIDs, 1)
		assert.Empty(t, m.entryIDs, "one-time scans have no recurring entry")
	})

	t.Run("re-running replaces the pending report", func(t *testing.T) {
		m, mocks := newTestScheduler(t)
		cfg := scanConfig("cfg1", "cfg-one", dailySchedule(0, 0))
		mocks.scanConfigDS.EXPECT().GetScanConfiguration(gomock.Any(), "cfg1").
			Return(cfg, true, nil).Times(2)

		m.ScheduleRescanReport("cfg1")
		require.Len(t, m.rescanEntryIDs, 1)
		first := m.rescanEntryIDs["cfg1"]

		m.ScheduleRescanReport("cfg1")
		require.Len(t, m.rescanEntryIDs, 1)
		assert.NotEqual(t, first, m.rescanEntryIDs["cfg1"], "the pending report was replaced, not duplicated")
	})

	t.Run("removing the configuration drops the pending report", func(t *testing.T) {
		m, mocks := newTestScheduler(t)
		cfg := scanConfig("cfg1", "cfg-one", dailySchedule(0, 0))
		mocks.scanConfigDS.EXPECT().GetScanConfiguration(gomock.Any(), "cfg1").
			Return(cfg, true, nil)

		m.UpsertScanConfig(cfg)
		m.ScheduleRescanReport("cfg1")
		require.Len(t, m.rescanEntryIDs, 1)

		m.RemoveScanConfig("cfg1")
		assert.Empty(t, m.entryIDs)
		assert.Empty(t, m.rescanEntryIDs)
	})

	t.Run("disabled feature schedules nothing", func(t *testing.T) {
		m, _ := newTestScheduler(t)
		t.Setenv(features.ComplianceDecoupledReports.EnvVar(), "false")
		// No GetScanConfiguration expectation: the disabled scheduler must not read the store.

		m.ScheduleRescanReport("cfg1")
		assert.Empty(t, m.rescanEntryIDs)
	})

	// End to end: the re-run report fires once, one delay after the latest re-run, and a
	// re-run before the report fires coalesces into it. Observation happens at the PREPARING
	// snapshot persist, which only a running report job performs (ScheduleRescanReport itself
	// also reads the configuration, so the read cannot distinguish the two).
	synctest.Test(t, func(t *testing.T) {
		m, mocks := newTestScheduler(t)
		mocks.scanConfigDS.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
		mocks.scanConfigDS.EXPECT().GetScanConfiguration(gomock.Any(), "cfg1").
			Return(scheduledScanConfig("cfg1", "cfg-one"), true, nil).AnyTimes()
		// Report data conversion for the generated report.
		mocks.scanConfigDS.EXPECT().GetScanConfigClusterStatus(gomock.Any(), "cfg1").Return(nil, nil).AnyTimes()
		mocks.suiteDS.EXPECT().GetSuites(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

		generated := make(chan struct{}, 4)
		mocks.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(), gomock.Any()).
			DoAndReturn(func(context.Context, *storage.ComplianceOperatorReportSnapshotV2) error {
				select {
				case generated <- struct{}{}:
				default:
				}
				return nil
			}).AnyTimes()
		mocks.reportGen.EXPECT().ProcessReportRequest(gomock.Any()).Return(nil).AnyTimes()

		m.start()
		defer m.Stop()

		firstScheduledAt := time.Now()
		m.ScheduleRescanReport("cfg1")

		// Half the delay: nothing has fired yet.
		select {
		case <-generated:
			t.Fatal("re-run report fired before the delay elapsed")
		case <-time.After(time.Until(firstScheduledAt.Add(30 * time.Minute))):
		}

		// Re-running coalesces: the report is due one delay after the latest re-run.
		secondScheduledAt := time.Now()
		m.ScheduleRescanReport("cfg1")

		// The first re-run's due time passes without a report.
		select {
		case <-generated:
			t.Fatal("re-run report fired at the first re-run's due time")
		case <-time.After(time.Until(firstScheduledAt.Add(59 * time.Minute))):
		}

		select {
		case <-generated:
		case <-time.After(time.Until(secondScheduledAt.Add(time.Hour + time.Minute))):
			t.Fatal("re-run report did not fire one delay after the latest re-run")
		}

		// The report fires exactly once.
		select {
		case <-generated:
			t.Fatal("re-run report fired twice")
		case <-time.After(time.Until(secondScheduledAt.Add(90 * time.Minute))):
		}
	})
}

// scheduledScanConfig returns a scan configuration that passes the scheduler's report
// conditions: it has notifiers and a last modifier.
func scheduledScanConfig(id, name string) *storage.ComplianceOperatorScanConfigurationV2 {
	cfg := scanConfig(id, name, dailySchedule(0, 0))
	cfg.Notifiers = []*storage.NotifierConfiguration{{}}
	cfg.ModifiedBy = &storage.SlimUser{Id: "user-1", Name: "User One"}
	return cfg
}

// TestRunReportJob covers the report job: it generates a SCHEDULED report from datastore
// state through the shared report machinery, skips configurations that cannot be reported,
// and fails the snapshot on cluster-data errors.
func TestRunReportJob(t *testing.T) {
	ctx := context.Background()

	t.Run("generates a scheduled report from datastore state", func(t *testing.T) {
		m, mocks := newTestScheduler(t)
		cfg := scheduledScanConfig("cfg1", "cfg-one")
		cfg.Profiles = []*storage.ComplianceOperatorScanConfigurationV2_ProfileName{{ProfileName: "ocp4"}}
		cfg.Clusters = []*storage.ComplianceOperatorScanConfigurationV2_Cluster{{ClusterId: "cluster-1"}}

		mocks.scanConfigDS.EXPECT().GetScanConfiguration(gomock.Any(), "cfg1").Return(cfg, true, nil)
		// Report data conversion: no cluster statuses, no suites.
		mocks.scanConfigDS.EXPECT().GetScanConfigClusterStatus(gomock.Any(), "cfg1").Return(nil, nil)
		mocks.suiteDS.EXPECT().GetSuites(gomock.Any(), gomock.Any()).Return(nil, nil)

		var persisted *storage.ComplianceOperatorReportSnapshotV2
		mocks.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, snapshot *storage.ComplianceOperatorReportSnapshotV2) error {
				persisted = snapshot
				return nil
			}).Times(1)

		mocks.reportGen.EXPECT().ProcessReportRequest(gomock.Any()).
			DoAndReturn(func(req *report.Request) error {
				assert.Equal(t, "cfg1", req.ScanConfigID)
				assert.Equal(t, "cfg-one", req.ScanConfigName)
				assert.Equal(t, []string{"ocp4"}, req.Profiles)
				assert.Equal(t, []string{"cluster-1"}, req.ClusterIDs)
				assert.Equal(t, storage.ComplianceOperatorReportStatus_EMAIL, req.NotificationMethod)
				assert.Equal(t, persisted.GetReportId(), req.SnapshotID)
				assert.Len(t, req.Notifiers, 1)
				assert.Equal(t, 0, req.NumFailedClusters)
				return nil
			}).Times(1)

		m.runReportJob(ctx, &reportJob{scanConfigID: "cfg1", scanConfigName: "cfg-one"})

		require.NotNil(t, persisted)
		assert.NotEmpty(t, persisted.GetReportId())
		assert.Equal(t, "cfg1", persisted.GetScanConfigurationId())
		assert.Equal(t, storage.ComplianceOperatorReportStatus_PREPARING, persisted.GetReportStatus().GetRunState())
		assert.Equal(t, storage.ComplianceOperatorReportStatus_SCHEDULED, persisted.GetReportStatus().GetReportRequestType())
		assert.Equal(t, storage.ComplianceOperatorReportStatus_EMAIL, persisted.GetReportStatus().GetReportNotificationMethod())
		assert.Equal(t, "user-1", persisted.GetUser().GetId())
	})

	t.Run("configuration not found is skipped", func(t *testing.T) {
		m, mocks := newTestScheduler(t)
		mocks.scanConfigDS.EXPECT().GetScanConfiguration(gomock.Any(), "cfg-gone").Return(nil, false, nil)
		// No snapshot or generator expectations: gomock fails the test if they are called.

		m.runReportJob(ctx, &reportJob{scanConfigID: "cfg-gone", scanConfigName: "cfg-gone"})
	})

	t.Run("configuration without notifiers is skipped", func(t *testing.T) {
		m, mocks := newTestScheduler(t)
		cfg := scanConfig("cfg1", "cfg-noemail", dailySchedule(0, 0))

		mocks.scanConfigDS.EXPECT().GetScanConfiguration(gomock.Any(), "cfg1").Return(cfg, true, nil)
		// No snapshot or generator expectations: nothing may be persisted or generated.

		m.runReportJob(ctx, &reportJob{scanConfigID: "cfg1", scanConfigName: "cfg-noemail"})
	})

	t.Run("cluster data error fails the snapshot", func(t *testing.T) {
		m, mocks := newTestScheduler(t)
		cfg := scheduledScanConfig("cfg1", "cfg-one")

		mocks.scanConfigDS.EXPECT().GetScanConfiguration(gomock.Any(), "cfg1").Return(cfg, true, nil)
		mocks.scanConfigDS.EXPECT().GetScanConfigClusterStatus(gomock.Any(), "cfg1").
			Return([]*storage.ComplianceOperatorClusterScanConfigStatus{{ClusterId: "cluster-1", ClusterName: "cluster-one"}}, nil)
		mocks.suiteDS.EXPECT().GetSuites(gomock.Any(), gomock.Any()).Return(nil, nil)
		mocks.bindingsDS.EXPECT().GetScanSettingBindings(gomock.Any(), gomock.Any()).Return(nil, nil)
		// GetClusterData resolves the cluster's scans and fails.
		mocks.scanDS.EXPECT().SearchScans(gomock.Any(), gomock.Any()).Return(nil, errors.New("boom"))

		var states []storage.ComplianceOperatorReportStatus_RunState
		mocks.snapshotDS.EXPECT().UpsertSnapshot(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, snapshot *storage.ComplianceOperatorReportSnapshotV2) error {
				states = append(states, snapshot.GetReportStatus().GetRunState())
				return nil
			}).Times(2)
		// No ProcessReportRequest expectation: the report must not be generated.

		m.runReportJob(ctx, &reportJob{scanConfigID: "cfg1", scanConfigName: "cfg-one"})

		require.Len(t, states, 2)
		assert.Equal(t, storage.ComplianceOperatorReportStatus_PREPARING, states[0])
		assert.Equal(t, storage.ComplianceOperatorReportStatus_FAILURE, states[1])
	})
}
