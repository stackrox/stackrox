package manager

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	snapshotDS "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore"
	scanConfigurationDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dblock"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

var schedulerLog = logging.LoggerForModule()

const (
	// defaultSchedulerRecheckInterval is how long a config scheduler sleeps
	// when nothing is due yet (it also bounds how quickly cron edits and
	// added notifiers are noticed).
	defaultSchedulerRecheckInterval = 15 * time.Minute
	// defaultSchedulerRetryInterval is the backoff for transient datastore
	// errors and failed report submissions.
	defaultSchedulerRetryInterval = 30 * time.Second
	// defaultSchedulerRespawnDelay is the backoff before respawning an
	// unexpectedly-exited scheduler, so a scheduler that dies repeatedly
	// cannot hot-spin.
	defaultSchedulerRespawnDelay = 30 * time.Second
	// defaultSchedulerMissingThreshold is how many consecutive not-found
	// reads a config scheduler tolerates before concluding that its scan
	// configuration is gone (single missed reads are treated as glitches).
	defaultSchedulerMissingThreshold = 3
)

// reportSchedulerManager owns one configScheduler per scan configuration
// (ROX_COMPLIANCE_REPORTS_OWN_SCHEDULER), decoupling scheduled reports from
// scan completion: each scheduler sleeps until the report for the current
// schedule fire is due (fire + ROX_COMPLIANCE_REPORT_SCHEDULE_DELAY) and
// submits it through the same settle-audit-snapshot primitive as on-demand
// reports.
//
// Lifecycle: the scan-config pipeline calls OnScanConfigUpsert /
// OnScanConfigDeleted on create/update/delete, a resync ticker reconciles
// against the datastore as a safety net for missed events, and a watcher
// goroutine per scheduler respawns it if it exits for any reason other than
// being commanded to stop. The map of running schedulers is the authority for
// that distinction: delete (and manager shutdown) removes the entry before
// stopping the scheduler, so a watcher that finds its scheduler's entry
// missing -- or replaced by a newer instance -- does not respawn.
//
// In HA deployments, start() acquires a PostgreSQL advisory lock so only one
// Central instance schedules reports (mirroring the vulnerability report
// scheduler).
type reportSchedulerManager struct {
	ctx          context.Context
	scanConfigDS scanConfigurationDS.DataStore
	snapshotDS   snapshotDS.DataStore
	// submit queues a scheduled report request for execution.
	submit func(scanConfig *storage.ComplianceOperatorScanConfigurationV2) error
	// db is used for the advisory lock; nil disables locking (tests).
	db postgres.DB

	delay            time.Duration
	recheckInterval  time.Duration
	retryInterval    time.Duration
	respawnDelay     time.Duration
	missingThreshold int
	resyncInterval   time.Duration

	// now and sleep are injectable for tests.
	now   func() time.Time
	sleep func(stop <-chan struct{}, d time.Duration) bool

	stopper           concurrency.Stopper
	isStarted         atomic.Bool
	isStopped         atomic.Bool
	releaseDb         func()
	initialResyncDone chan struct{}

	mu sync.Mutex
	// schedulers maps scan configuration IDs to their running scheduler.
	// Guarded by mu; also the authority for "was this exit commanded?" (see
	// watch). scheduling is only mutated under mu as well, so spawn decisions
	// and shutdown are serialized.
	schedulers map[string]*configScheduler
	scheduling bool
}

func newReportSchedulerManager(ctx context.Context, scanConfigDS scanConfigurationDS.DataStore, snapshotDS snapshotDS.DataStore,
	submit func(scanConfig *storage.ComplianceOperatorScanConfigurationV2) error, db postgres.DB) *reportSchedulerManager {
	return &reportSchedulerManager{
		ctx:              ctx,
		scanConfigDS:     scanConfigDS,
		snapshotDS:       snapshotDS,
		submit:           submit,
		db:               db,
		delay:            env.ComplianceReportScheduleDelay.DurationSetting(),
		recheckInterval:  defaultSchedulerRecheckInterval,
		retryInterval:    defaultSchedulerRetryInterval,
		respawnDelay:     defaultSchedulerRespawnDelay,
		missingThreshold: defaultSchedulerMissingThreshold,
		resyncInterval:   env.ComplianceReportSchedulerResyncInterval.DurationSetting(),
		now:              time.Now,
		sleep:            sleepUntilStop,
		stopper:          concurrency.NewStopper(),
		// initialResyncDone is closed by run() after the startup resync; it
		// lets callers (and tests) synchronize with the asynchronous seeding.
		initialResyncDone: make(chan struct{}),
		schedulers:        make(map[string]*configScheduler),
	}
}

// start starts scheduling reports. In HA deployments it first acquires the
// compliance report scheduler advisory lock; if another Central holds it,
// this instance does not schedule (the pipeline hooks become no-ops).
func (m *reportSchedulerManager) start() {
	if m.isStopped.Load() {
		schedulerLog.Error("Report scheduler manager already stopped. It cannot be re-started once stopped.")
		return
	}
	if !m.isStarted.CompareAndSwap(false, true) {
		schedulerLog.Error("Report scheduler manager already running")
		return
	}
	if m.db != nil {
		acquired, release, err := dblock.TryAcquireAdvisoryLock(m.ctx, m.db, dblock.ComplianceReportSchedulerLockID)
		if err != nil {
			schedulerLog.Errorf("Report scheduler manager: failed to acquire the advisory lock, not scheduling: %v", err)
			m.isStarted.Store(false)
			return
		}
		if !acquired {
			schedulerLog.Info("Report scheduler manager: advisory lock held by another Central, not scheduling")
			m.isStarted.Store(false)
			return
		}
		m.releaseDb = release
	}
	concurrency.WithLock(&m.mu, func() {
		m.scheduling = true
	})
	go m.run()
	schedulerLog.Info("Report scheduler manager started")
}

// stop stops the manager: the resync loop first, then every running scheduler.
// Map entries are removed before the schedulers are stopped, so their watchers
// do not respawn them.
func (m *reportSchedulerManager) stop() {
	if !m.isStarted.Load() {
		return
	}
	if !m.isStopped.CompareAndSwap(false, true) {
		return
	}
	m.stopper.Client().Stop()
	if err := m.stopper.Client().Stopped().Wait(); err != nil {
		schedulerLog.Errorf("Error stopping the report scheduler manager: %v", err)
	}
	if m.releaseDb != nil {
		m.releaseDb()
	}

	schedulers := concurrency.WithLock1[[]*configScheduler](&m.mu, func() []*configScheduler {
		m.scheduling = false
		ret := make([]*configScheduler, 0, len(m.schedulers))
		for _, s := range m.schedulers {
			ret = append(ret, s)
		}
		m.schedulers = make(map[string]*configScheduler)
		return ret
	})
	for _, s := range schedulers {
		s.stopper.Client().Stop()
	}
	schedulerLog.Info("Report scheduler manager stopped")
}

// run seeds the schedulers from the datastore and then reconciles them on
// every resync tick. The resync is the safety net for missed pipeline events
// (and rebuilds the set after a lock failover); the primary triggers are the
// pipeline hooks and the schedulers' own due-time wake-ups.
func (m *reportSchedulerManager) run() {
	defer m.stopper.Flow().ReportStopped()
	m.resync()
	close(m.initialResyncDone)
	ticker := time.NewTicker(m.resyncInterval)
	defer ticker.Stop()
	for {
		select {
		case <-m.stopper.Flow().StopRequested():
			return
		case <-ticker.C:
			m.resync()
		}
	}
}

// resync reconciles the running schedulers with the scan configurations in
// the datastore: spawns schedulers for new configurations and stops
// schedulers whose configuration is gone.
func (m *reportSchedulerManager) resync() {
	scanConfigs, err := m.scanConfigDS.GetScanConfigurations(m.ctx, search.EmptyQuery())
	if err != nil {
		schedulerLog.Errorf("Report scheduler manager: unable to retrieve scan configurations: %v", err)
		return
	}
	active := set.NewStringSet()
	for _, scanConfig := range scanConfigs {
		active.Add(scanConfig.GetId())
		m.onScanConfigUpsert(scanConfig)
	}

	stale := concurrency.WithLock1[[]*configScheduler](&m.mu, func() []*configScheduler {
		var ret []*configScheduler
		for id, s := range m.schedulers {
			if !active.Contains(id) {
				delete(m.schedulers, id)
				ret = append(ret, s)
			}
		}
		return ret
	})
	for _, s := range stale {
		schedulerLog.Infof("Report scheduler manager: stopping the scheduler of removed scan configuration %q", s.id)
		s.stopper.Client().Stop()
	}
}

// onScanConfigUpsert is called by the scan-config pipeline when a scan
// configuration is created or updated. Updates need no restart: the scheduler
// re-reads the configuration on every wake-up, so cron edits and notifier
// changes are picked up by the running scheduler.
func (m *reportSchedulerManager) onScanConfigUpsert(scanConfig *storage.ComplianceOperatorScanConfigurationV2) {
	if scanConfig == nil {
		return
	}
	concurrency.WithLock(&m.mu, func() {
		m.spawnLocked(scanConfig.GetId())
	})
}

// onScanConfigDeleted is called by the scan-config pipeline when a scan
// configuration is deleted. The scheduler is removed from the map before
// being stopped, so its watcher does not respawn it.
func (m *reportSchedulerManager) onScanConfigDeleted(scanConfigID string) {
	s := concurrency.WithLock1[*configScheduler](&m.mu, func() *configScheduler {
		s := m.schedulers[scanConfigID]
		delete(m.schedulers, scanConfigID)
		return s
	})
	if s != nil {
		s.stopper.Client().Stop()
	}
}

// spawnLocked starts the scheduler of the given scan configuration unless a
// live scheduler is already running for it. The entry of a scheduler that
// already reported stopped -- an unexpectedly-exited scheduler whose respawn
// is still in its backoff -- is replaced, so a dead entry can never block the
// spawn. Callers hold mu.
func (m *reportSchedulerManager) spawnLocked(id string) {
	if !m.scheduling {
		return
	}
	if existing, running := m.schedulers[id]; running && !existing.stopper.Client().Stopped().IsDone() {
		return
	}
	s := &configScheduler{
		id:               id,
		ctx:              m.ctx,
		scanConfigDS:     m.scanConfigDS,
		snapshotDS:       m.snapshotDS,
		submit:           m.submit,
		delay:            m.delay,
		recheckInterval:  m.recheckInterval,
		retryInterval:    m.retryInterval,
		missingThreshold: m.missingThreshold,
		now:              m.now,
		sleep:            m.sleep,
		stopper:          concurrency.NewStopper(),
	}
	m.schedulers[id] = s
	go s.Run()
	go m.watch(id, s)
	schedulerLog.Infof("Report scheduler manager: spawned the scheduler of scan configuration %q", id)
}

// watch waits for the scheduler to stop and respawns it if the exit was not
// commanded. The map is the authority for that distinction: delete and
// manager shutdown remove (or replace) the entry before the scheduler stops,
// so a watcher that no longer finds its own scheduler instance in the map
// does not respawn. A scheduler that reports errScanConfigGone deregisters
// itself instead. Unexpected exits are retried after a backoff, so a
// scheduler that dies repeatedly cannot hot-spin.
func (m *reportSchedulerManager) watch(id string, s *configScheduler) {
	stopped := s.stopper.Client().Stopped()
	<-stopped.Done()
	err := stopped.Err()

	if !errors.Is(err, errScanConfigGone) {
		// Back off before respawning. Delete and shutdown remove the map
		// entry; the check after the backoff catches it. The backoff is
		// interrupted when the manager itself stops.
		if !m.sleep(m.stopper.LowLevel().GetStopRequestSignal().Done(), m.respawnDelay) {
			return
		}
	}

	concurrency.WithLock(&m.mu, func() {
		current, running := m.schedulers[id]
		if !running || current != s {
			// The exit was commanded (the entry was removed or replaced by a
			// newer scheduler): do not respawn.
			return
		}
		if errors.Is(err, errScanConfigGone) {
			// The scheduler observed the configuration disappear; deregister
			// instead of respawning.
			delete(m.schedulers, id)
			return
		}
		schedulerLog.Warnf("Report scheduler for scan configuration %q exited unexpectedly (%v); respawning", id, err)
		m.spawnLocked(id)
	})
}
