package manager

import (
	"context"
	"sync"
	"testing"
	"time"

	snapshotMocks "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore/mocks"
	scanConfigurationDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// schedulerManagerTest wraps a reportSchedulerManager on freshly created
// mocks. The scan configurations "in the database" are controlled through
// dbConfigs (served to resync) and the per-ID getConfig function (served to
// the schedulers' loops). By default every served configuration's report is
// due far in the future, so spawned schedulers park in their sleep instead of
// submitting, and exits are driven by the stoppers.
type schedulerManagerTest struct {
	*reportSchedulerManager
	ctrl      *gomock.Controller
	dbConfigs []*storage.ComplianceOperatorScanConfigurationV2
	getConfig func(id string) (*storage.ComplianceOperatorScanConfigurationV2, bool, error)
	submitted []string
	submitMu  sync.Mutex
}

func newSchedulerManagerTest(t *testing.T) *schedulerManagerTest {
	t.Helper()
	ctrl := gomock.NewController(t)
	scanConfigDS := scanConfigurationDS.NewMockDataStore(ctrl)
	snapshotDS := snapshotMocks.NewMockDataStore(ctrl)

	ts := &schedulerManagerTest{ctrl: ctrl}
	ts.getConfig = func(id string) (*storage.ComplianceOperatorScanConfigurationV2, bool, error) {
		// A future fire: the schedule fires 6 hours from now, and with the
		// 24h delay below the report is always due in the future.
		sc := loopTestConfig(id)
		fire := time.Now().Add(6 * time.Hour)
		sc.Schedule = &storage.Schedule{
			IntervalType: storage.Schedule_DAILY,
			Hour:         int32(fire.Hour()),
			Minute:       int32(fire.Minute()),
		}
		return sc, true, nil
	}

	scanConfigDS.EXPECT().GetScanConfigurations(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(context.Context, any) ([]*storage.ComplianceOperatorScanConfigurationV2, error) {
			configs := make([]*storage.ComplianceOperatorScanConfigurationV2, len(ts.dbConfigs))
			copy(configs, ts.dbConfigs)
			return configs, nil
		})
	scanConfigDS.EXPECT().GetScanConfiguration(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_ context.Context, id string) (*storage.ComplianceOperatorScanConfigurationV2, bool, error) {
			return ts.getConfig(id)
		})
	snapshotDS.EXPECT().SearchSnapshots(gomock.Any(), gomock.Any()).AnyTimes().
		Return(nil, nil)

	ts.reportSchedulerManager = &reportSchedulerManager{
		ctx:          context.Background(),
		scanConfigDS: scanConfigDS,
		snapshotDS:   snapshotDS,
		submit: func(scanConfig *storage.ComplianceOperatorScanConfigurationV2) error {
			ts.submitMu.Lock()
			defer ts.submitMu.Unlock()
			ts.submitted = append(ts.submitted, scanConfig.GetId())
			return nil
		},
		db:               nil,
		delay:            24 * time.Hour, // reports are never due in these tests unless overridden
		recheckInterval:  time.Hour,
		retryInterval:    time.Hour,
		respawnDelay:     0, // watch backoff is a no-op
		missingThreshold: 3,
		resyncInterval:   time.Hour, // no resync ticks; tests drive resync manually
		now:              time.Now,
		sleep:            sleepUntilStop,
		stopper:          concurrency.NewStopper(),
		// initialResyncDone is closed by run() after the startup resync; it
		// lets tests synchronize with the asynchronous seeding.
		initialResyncDone: make(chan struct{}),
		schedulers:        make(map[string]*configScheduler),
	}
	t.Cleanup(func() {
		ts.reportSchedulerManager.stop()
		ctrl.Finish()
	})
	return ts
}

func (ts *schedulerManagerTest) schedulerCount() int {
	return concurrency.WithLock1[int](&ts.mu, func() int {
		return len(ts.schedulers)
	})
}

// startAndWait starts the manager and waits for its startup resync to
// complete, so test mutations of the database state and the scheduler map
// cannot race the asynchronous seeding.
func (ts *schedulerManagerTest) startAndWait() {
	ts.start()
	<-ts.initialResyncDone
}

func (ts *schedulerManagerTest) schedulerOf(id string) *configScheduler {
	return concurrency.WithLock1[*configScheduler](&ts.mu, func() *configScheduler {
		return ts.schedulers[id]
	})
}

func (ts *schedulerManagerTest) submittedIDs() []string {
	ts.submitMu.Lock()
	defer ts.submitMu.Unlock()
	return append([]string(nil), ts.submitted...)
}

// TestSchedulerManagerUpsertSpawnsOnce verifies that pipeline upsert events
// spawn exactly one scheduler per scan configuration.
func TestSchedulerManagerUpsertSpawnsOnce(t *testing.T) {
	ts := newSchedulerManagerTest(t)
	ts.startAndWait()

	ts.onScanConfigUpsert(loopTestConfig("config-1"))
	assert.Equal(t, 1, ts.schedulerCount())

	// A second upsert (e.g. an update) must not double-spawn.
	ts.onScanConfigUpsert(loopTestConfig("config-1"))
	assert.Equal(t, 1, ts.schedulerCount())
}

// TestSchedulerManagerDeleteStopsScheduler verifies that a delete event
// removes the scheduler from the map before stopping it, so the watcher does
// not respawn it.
func TestSchedulerManagerDeleteStopsScheduler(t *testing.T) {
	ts := newSchedulerManagerTest(t)
	ts.startAndWait()
	ts.onScanConfigUpsert(loopTestConfig("config-1"))
	s := ts.schedulerOf("config-1")
	require.NotNil(t, s)

	ts.onScanConfigDeleted("config-1")
	assert.Equal(t, 0, ts.schedulerCount(), "the map entry is removed synchronously")

	// The scheduler stops, and the watcher (backoff 0) must not respawn it.
	require.Eventually(t, func() bool { return s.stopper.Client().Stopped().IsDone() }, 2*time.Second, 5*time.Millisecond,
		"the scheduler should stop")
	time.Sleep(100 * time.Millisecond) // let the watcher process the exit
	assert.Equal(t, 0, ts.schedulerCount(), "a commanded stop must not be respawned")
}

// TestSchedulerManagerRespawnsUnexpectedExit verifies the supervision: a
// scheduler that stops without being commanded (the map entry still present)
// is respawned.
func TestSchedulerManagerRespawnsUnexpectedExit(t *testing.T) {
	ts := newSchedulerManagerTest(t)
	ts.startAndWait()
	ts.onScanConfigUpsert(loopTestConfig("config-1"))
	s1 := ts.schedulerOf("config-1")
	require.NotNil(t, s1)

	// Simulate an unexpected exit without removing the map entry.
	s1.stopper.Client().Stop()

	require.Eventually(t, func() bool {
		return ts.schedulerOf("config-1") != nil && ts.schedulerOf("config-1") != s1
	}, 2*time.Second, 5*time.Millisecond, "the scheduler should be respawned")
	assert.Equal(t, 1, ts.schedulerCount())

	// The respawned scheduler keeps running (parked on its future-due sleep).
	s2 := ts.schedulerOf("config-1")
	require.NotNil(t, s2)
	assert.False(t, s2.stopper.Client().Stopped().IsDone(), "the respawned scheduler should be running")
}

// TestSchedulerManagerDeregistersGoneConfig verifies that a scheduler whose
// scan configuration disappears (a delete event the manager missed) exits
// with errScanConfigGone and deregisters instead of respawning.
func TestSchedulerManagerDeregistersGoneConfig(t *testing.T) {
	ts := newSchedulerManagerTest(t)
	ts.retryInterval = 5 * time.Millisecond
	// The configuration is not in the datastore, and no delete event reaches
	// the manager. (Set before starting: the schedulers read it
	// concurrently.)
	ts.getConfig = func(string) (*storage.ComplianceOperatorScanConfigurationV2, bool, error) {
		return nil, false, nil
	}
	ts.startAndWait()
	ts.onScanConfigUpsert(loopTestConfig("config-1"))
	require.Equal(t, 1, ts.schedulerCount())

	require.Eventually(t, func() bool { return ts.schedulerCount() == 0 }, 2*time.Second, 5*time.Millisecond,
		"the scheduler should deregister itself after the configuration disappears")
}

// TestSchedulerManagerStaleWatcherDoesNotRespawn verifies the identity check:
// a watcher whose scheduler was replaced in the map does not respawn.
func TestSchedulerManagerStaleWatcherDoesNotRespawn(t *testing.T) {
	ts := newSchedulerManagerTest(t)
	ts.startAndWait()
	ts.onScanConfigUpsert(loopTestConfig("config-1"))
	s1 := ts.schedulerOf("config-1")
	require.NotNil(t, s1)

	// Replace the map entry with another scheduler instance, as a respawn or
	// a resync re-spawn would, and stop the old one.
	s2 := &configScheduler{id: "config-1", stopper: concurrency.NewStopper()}
	concurrency.WithLock(&ts.mu, func() {
		ts.schedulers["config-1"] = s2
	})
	s1.stopper.Client().Stop()
	require.Eventually(t, func() bool { return s1.stopper.Client().Stopped().IsDone() }, 2*time.Second, 5*time.Millisecond)

	// The stale watcher processes the exit of s1 (backoff 0) and must leave
	// the replacement alone.
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, s2, ts.schedulerOf("config-1"))
	assert.Equal(t, 1, ts.schedulerCount())
}

// TestSchedulerManagerResync verifies the safety net: resync spawns
// schedulers for configurations missing from the map and stops schedulers
// whose configuration is gone from the datastore.
func TestSchedulerManagerResync(t *testing.T) {
	ts := newSchedulerManagerTest(t)
	ts.startAndWait()
	ts.onScanConfigUpsert(loopTestConfig("config-1"))
	s1 := ts.schedulerOf("config-1")
	require.NotNil(t, s1)

	// The database now only knows config-2.
	ts.dbConfigs = []*storage.ComplianceOperatorScanConfigurationV2{loopTestConfig("config-2")}
	ts.resync()

	require.Eventually(t, func() bool { return ts.schedulerOf("config-2") != nil }, 2*time.Second, 5*time.Millisecond,
		"resync spawns the missing scheduler")
	require.Eventually(t, func() bool { return s1.stopper.Client().Stopped().IsDone() }, 2*time.Second, 5*time.Millisecond,
		"resync stops the scheduler of the removed configuration")
	time.Sleep(100 * time.Millisecond) // let the watcher process the exit
	assert.Equal(t, 1, ts.schedulerCount())
	assert.Nil(t, ts.schedulerOf("config-1"))
}

// TestSchedulerManagerHooksRequireScheduling verifies that pipeline hooks are
// no-ops unless the manager is scheduling (feature disabled, or another
// Central holds the advisory lock).
func TestSchedulerManagerHooksRequireScheduling(t *testing.T) {
	ts := newSchedulerManagerTest(t)
	// Not started: scheduling is false.
	ts.onScanConfigUpsert(loopTestConfig("config-1"))
	assert.Equal(t, 0, ts.schedulerCount())
}

// TestSchedulerManagerStartStopSeedsAndDrains verifies the startup seeding
// from the database and the shutdown drain.
func TestSchedulerManagerStartStopSeedsAndDrains(t *testing.T) {
	ts := newSchedulerManagerTest(t)
	ts.dbConfigs = []*storage.ComplianceOperatorScanConfigurationV2{loopTestConfig("config-1")}
	ts.start()

	require.Eventually(t, func() bool { return ts.schedulerCount() == 1 }, 2*time.Second, 5*time.Millisecond,
		"the startup resync seeds the schedulers from the database")
	s := ts.schedulerOf("config-1")
	require.NotNil(t, s)

	ts.stop()
	assert.Equal(t, 0, ts.schedulerCount(), "stop drains the map")
	require.Eventually(t, func() bool { return s.stopper.Client().Stopped().IsDone() }, 2*time.Second, 5*time.Millisecond,
		"stop stops the running schedulers")
	time.Sleep(100 * time.Millisecond) // let the watcher process the exit
	assert.Equal(t, 0, ts.schedulerCount(), "stopped schedulers are not respawned")
}

// TestSchedulerManagerSubmitsDueReport verifies the end-to-end path: a
// spawned scheduler submits a due report through the submit callback.
func TestSchedulerManagerSubmitsDueReport(t *testing.T) {
	ts := newSchedulerManagerTest(t)
	ts.delay = 0 // reports are due immediately
	ts.startAndWait()
	ts.onScanConfigUpsert(loopTestConfig("config-1"))

	require.Eventually(t, func() bool {
		return len(ts.submittedIDs()) == 1
	}, 2*time.Second, 5*time.Millisecond, "the scheduler should submit the due report")
	assert.Equal(t, []string{"config-1"}, ts.submittedIDs())
}
