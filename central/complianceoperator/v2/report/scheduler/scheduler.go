// Package scheduler contains the report scheduler of the decoupled compliance reporting
// redesign (see compliance-report-partial-cycles.md, proposal 1).
//
// Reports are scheduled per scan configuration, decoupled from scan completion: every scan
// configuration with an evaluable schedule gets a cron entry whose report job fires at each of
// its schedule ticks plus a fixed delay (ROX_COMPLIANCE_REPORT_SCHEDULE_DELAY). The design
// mirrors the vulnerability-management report scheduler (central/reports/scheduler/v2) with
// compliance-typed jobs: a shared cron instance computes exact per-configuration fire times
// with a single goroutine, entries submit jobs to a queue rather than running them, and a
// signal-driven execution loop runs the jobs with bounded concurrency and one job per
// configuration at a time. Anything that is not a scheduled recurrence (on-demand reports,
// re-run-triggered reports) submits jobs through the same queue without touching the cron.
//
// The scheduler is gated by ROX_COMPLIANCE_DECOUPLED_REPORTS and holds a PostgreSQL advisory
// lock while running, so only one Central schedules reports in HA deployments.
//
// Report jobs generate and deliver reports from the current datastore state through the same
// shared building blocks as the report manager (helpers.* and the report generator), so the
// UI renders their snapshots like any other report. Settling scans that are still running at
// report time and auditing schedule expectations land in later iterations of the rollout.
package scheduler

import (
	"context"
	"sync/atomic"
	"time"

	integrationDS "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	snapshotDS "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore"
	reportGen "github.com/stackrox/rox/central/complianceoperator/v2/report/manager/generator"
	scanConfigDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	scanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	bindingsDS "github.com/stackrox/rox/central/complianceoperator/v2/scansettingbindings/datastore"
	suiteDS "github.com/stackrox/rox/central/complianceoperator/v2/suites/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dblock"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"golang.org/x/sync/semaphore"
	"gopkg.in/robfig/cron.v2"
)

var log = logging.LoggerForModule()

// Manager schedules report jobs for compliance scan configurations, decoupled from scan
// completion: the report job of a configuration fires on the configuration's own schedule
// (its scan schedule shifted by a fixed delay), not when its scans complete.
//
//go:generate mockgen-wrapper
type Manager interface {
	// Start acquires the report scheduler advisory lock and starts the scheduler: it
	// reconciles one cron entry per stored scan configuration with an evaluable schedule and
	// runs submitted report jobs. Start is idempotent and a no-op while the
	// ROX_COMPLIANCE_DECOUPLED_REPORTS feature is disabled.
	Start(db postgres.DB)
	// Stop stops the scheduler, releases its cron entries, and releases the advisory lock.
	// Stop is idempotent; a stopped scheduler cannot be restarted.
	Stop()
	// UpsertScanConfig creates or updates the cron entry of the given scan configuration,
	// e.g. on configuration create or update. Configurations without an evaluable schedule
	// (one-time scans) get no entry; upserting one removes any entry the configuration had.
	UpsertScanConfig(scanConfig *storage.ComplianceOperatorScanConfigurationV2)
	// RemoveScanConfig drops the cron entries of the scan configuration, if any, e.g. on
	// configuration deletion. This includes a pending one-shot report.
	RemoveScanConfig(scanConfigID string)
	// ScheduleRescanReport schedules a one-shot report job for the scan configuration, due one
	// delay after now: a re-run's scans get the delay to settle before the report is
	// generated. Re-running again before the report fires coalesces into a single job due one
	// delay after the latest re-run. One-time scans are included: a re-run is their only cycle.
	ScheduleRescanReport(scanConfigID string)
	// ScheduleFirstScanReport schedules a one-shot report job for the scan configuration's
	// first scan cycle, due one delay after creation, so the creation-triggered scans are
	// reported even when the recurring entry's first fire is a full schedule interval away.
	// One-time scans are included: the first cycle is their only report.
	ScheduleFirstScanReport(scanConfig *storage.ComplianceOperatorScanConfigurationV2)
	// SubmitReportRequest submits an on-demand report execution request for the scan
	// configuration. The report snapshot is persisted in the WAITING state immediately, so the
	// UI shows it right away, and the report job runs through the same execution loop as
	// scheduled reports. A second request for a configuration with a pending on-demand report
	// is rejected, mirroring the report manager.
	SubmitReportRequest(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2, method storage.ComplianceOperatorReportStatus_NotificationMethod) error
}

type managerImpl struct {
	scanConfigDataStore  scanConfigDS.DataStore
	scanDataStore        scanDS.DataStore
	snapshotDataStore    snapshotDS.DataStore
	suiteDataStore       suiteDS.DataStore
	bindingsDataStore    bindingsDS.DataStore
	integrationDataStore integrationDS.DataStore
	reportGenerator      reportGen.ComplianceReportGenerator

	stopper   concurrency.Stopper
	isStarted atomic.Bool
	isStopped atomic.Bool

	// stopCtx is cancelled once shutdown is requested; it unblocks the execution loop's
	// concurrency acquisition.
	stopCtx context.Context

	// schedulerCtx scopes the scheduler's background work: no user identity, Compliance access
	// only (as the freshness sweeper). It is cancelled once shutdown is requested, so neither
	// datastore reads nor running jobs outlive Stop().
	schedulerCtx context.Context

	// cron triggers one entry per scan configuration; entries submit jobs rather than running
	// them, so the cron goroutine never blocks on report generation.
	cron *cron.Cron
	// cronJobsLock guards entryIDs and oneShotEntryIDs.
	cronJobsLock sync.Mutex
	// entryIDs maps scan configuration IDs to their recurring schedule entries.
	entryIDs map[string]cron.EntryID
	// oneShotEntryIDs maps scan configuration IDs to their pending one-shot entries (re-run
	// and first-scan reports).
	oneShotEntryIDs map[string]cron.EntryID

	// jobQueue holds submitted report jobs; readyForReports wakes the execution loop when a
	// job is enqueued or a running one finishes.
	jobQueue        *jobQueue
	readyForReports concurrency.Signal
	concurrencySem  *semaphore.Weighted

	// onDemandLock guards onDemandPending.
	onDemandLock sync.Mutex
	// onDemandPending holds the scan configurations with a submitted on-demand report that
	// has not finished yet; a second request for the same configuration is rejected.
	onDemandPending set.StringSet

	// advisoryLockRelease releases the report scheduler advisory lock, if held.
	advisoryLockRelease func()
}

// New returns a report scheduler backed by the given scan configuration datastore and report
// generation dependencies.
func New(scanConfigDataStore scanConfigDS.DataStore,
	scanDataStore scanDS.DataStore,
	snapshotDataStore snapshotDS.DataStore,
	suiteDataStore suiteDS.DataStore,
	bindingsDataStore bindingsDS.DataStore,
	integrationDataStore integrationDS.DataStore,
	reportGenerator reportGen.ComplianceReportGenerator) Manager {
	stopper := concurrency.NewStopper()
	stopCtx := concurrency.AsContext(stopper.LowLevel().GetStopRequestSignal())
	m := &managerImpl{
		scanConfigDataStore:  scanConfigDataStore,
		scanDataStore:        scanDataStore,
		snapshotDataStore:    snapshotDataStore,
		suiteDataStore:       suiteDataStore,
		bindingsDataStore:    bindingsDataStore,
		integrationDataStore: integrationDataStore,
		reportGenerator:      reportGenerator,
		stopper:              stopper,
		stopCtx:              stopCtx,
		schedulerCtx: sac.WithGlobalAccessScopeChecker(
			stopCtx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resources.Compliance),
			),
		),
		cron:            cron.New(),
		entryIDs:        make(map[string]cron.EntryID),
		oneShotEntryIDs: make(map[string]cron.EntryID),
		jobQueue:        newJobQueue(),
		readyForReports: concurrency.NewSignal(),
		concurrencySem:  semaphore.NewWeighted(int64(env.ReportExecutionMaxConcurrency.IntegerSetting())),
		onDemandPending: set.NewStringSet(),
	}
	return m
}

// enabled reports whether the decoupled report scheduler is active: the rollout flag requires
// the compliance reporting feature it belongs to.
func (m *managerImpl) enabled() bool {
	return features.ComplianceReporting.Enabled() && features.ComplianceDecoupledReports.Enabled()
}

// Start acquires the compliance report scheduler advisory lock and starts the scheduler. If
// the lock is already held by another process (HA deployment), the scheduler is not started.
// A scheduler instance can only be started once and cannot be re-started once stopped.
func (m *managerImpl) Start(db postgres.DB) {
	if !m.enabled() {
		log.Info("Compliance report scheduler: decoupled reporting is disabled, not starting")
		return
	}
	acquired, release, err := dblock.TryAcquireAdvisoryLock(m.stopCtx, db, dblock.ComplianceReportSchedulerLockID)
	if err != nil {
		log.Errorf("Compliance report scheduler: failed to acquire the advisory lock: %v", err)
		return
	}
	if !acquired {
		log.Info("Compliance report scheduler: advisory lock held by another process, not starting")
		return
	}
	if !m.start() {
		// The scheduler was already running or stopped; release what this call acquired.
		release()
		return
	}
	m.advisoryLockRelease = release
}

// start starts the scheduler if it has neither started before nor been stopped, and reports
// whether it did. It is the test and lock-free entry point.
func (m *managerImpl) start() bool {
	if m.isStopped.Load() {
		log.Error("Compliance report scheduler already stopped. It cannot be re-started once stopped.")
		return false
	}
	if !m.isStarted.CompareAndSwap(false, true) {
		log.Error("Compliance report scheduler already running")
		return false
	}
	log.Info("Starting compliance report scheduler")

	// Reconcile before the cron starts running so stored configurations get entries without
	// producing events (Central restarts produce none). Entries added while the cron is not
	// running are parked and scheduled when it starts.
	m.reconcile(m.schedulerCtx)
	m.cron.Start()
	go m.runJobs()
	return true
}

func (m *managerImpl) Stop() {
	if !m.isStarted.Load() {
		log.Error("Compliance report scheduler not started")
		return
	}
	if !m.isStopped.CompareAndSwap(false, true) {
		log.Error("Compliance report scheduler already stopped")
		return
	}
	log.Info("Stopping compliance report scheduler")
	m.cron.Stop()
	m.stopper.Client().Stop()
	if err := m.stopper.Client().Stopped().Wait(); err != nil {
		log.Errorf("Error stopping the compliance report scheduler: %v", err)
	}
	concurrency.WithLock(&m.cronJobsLock, func() {
		m.entryIDs = make(map[string]cron.EntryID)
		m.oneShotEntryIDs = make(map[string]cron.EntryID)
	})
	if m.advisoryLockRelease != nil {
		m.advisoryLockRelease()
	}
}

// runJobs is the execution loop: it runs one submitted report job per signal, bounded by the
// concurrency semaphore, until no job is runnable; the signal is reset then. A finishing job
// re-signals, so a job that waited behind a running one for the same configuration is picked
// up without a new submission.
func (m *managerImpl) runJobs() {
	defer m.stopper.Flow().ReportStopped()
	for {
		select {
		case <-m.stopper.Flow().StopRequested():
			return
		case <-m.readyForReports.Done():
			// Hold execution capacity for the next job; shutdown cancels the acquisition.
			if err := m.concurrencySem.Acquire(m.stopCtx, 1); err != nil {
				// Shutdown cancelled the context; retrying cannot acquire capacity.
				return
			}
			select {
			case <-m.stopper.Flow().StopRequested():
				m.concurrencySem.Release(1)
				return
			default:
			}
			job := m.jobQueue.dequeue()
			if job == nil {
				m.concurrencySem.Release(1)
				m.readyForReports.Reset()
				continue
			}
			go m.runSingleJob(job)
		}
	}
}

func (m *managerImpl) runSingleJob(job *reportJob) {
	defer m.readyForReports.Signal()
	defer m.concurrencySem.Release(1)
	// Free the configuration only once this job is done, so a job that arrives while one is
	// running waits in the queue rather than colliding with it.
	defer m.jobQueue.markDone(job.scanConfigID)
	// Free the on-demand submission slot once the report is done, so a new request for the
	// configuration is accepted.
	defer m.onDemandDone(job)

	m.runReportJob(m.schedulerCtx, job)
}

// onDemandDone releases the configuration's on-demand submission slot after its report job
// finished, successfully or not.
func (m *managerImpl) onDemandDone(job *reportJob) {
	if job.snapshotID == "" {
		return
	}
	concurrency.WithLock(&m.onDemandLock, func() {
		m.onDemandPending.Remove(job.scanConfigID)
	})
}

// submitJob enqueues a report job and wakes the execution loop. Cron entries and on-demand
// submissions go through here.
func (m *managerImpl) submitJob(job *reportJob) {
	m.jobQueue.enqueue(job)
	m.readyForReports.Signal()
}

// reconcile rebuilds the entry set from the stored scan configurations so that exactly the
// configurations with an evaluable schedule have a cron entry.
func (m *managerImpl) reconcile(ctx context.Context) {
	configs, err := m.scanConfigDataStore.GetScanConfigurations(ctx, search.EmptyQuery())
	if err != nil {
		log.Errorf("Compliance report scheduler: unable to list scan configurations: %v", err)
		return
	}
	concurrency.WithLock(&m.cronJobsLock, func() {
		withEntry := set.NewStringSet()
		for _, cfg := range configs {
			m.upsertScanConfigNoLock(cfg)
			if _, ok := m.entryIDs[cfg.GetId()]; ok {
				withEntry.Add(cfg.GetId())
			}
		}
		for id := range m.entryIDs {
			if !withEntry.Contains(id) {
				m.removeScanConfigNoLock(id)
			}
		}
	})
}

func (m *managerImpl) UpsertScanConfig(scanConfig *storage.ComplianceOperatorScanConfigurationV2) {
	if !m.enabled() {
		return
	}
	concurrency.WithLock(&m.cronJobsLock, func() {
		m.upsertScanConfigNoLock(scanConfig)
	})
}

func (m *managerImpl) RemoveScanConfig(scanConfigID string) {
	if !m.enabled() {
		return
	}
	concurrency.WithLock(&m.cronJobsLock, func() {
		m.removeScanConfigNoLock(scanConfigID)
	})
}

// ScheduleRescanReport schedules a one-shot report job for the scan configuration, due one
// delay after now. A pending one-shot report is replaced, so repeated re-runs coalesce into
// one report due one delay after the latest re-run.
func (m *managerImpl) ScheduleRescanReport(scanConfigID string) {
	if !m.enabled() {
		return
	}
	scanConfig, found, err := m.scanConfigDataStore.GetScanConfiguration(m.schedulerCtx, scanConfigID)
	if err != nil {
		log.Errorf("Compliance report scheduler: unable to retrieve scan configuration %q for the re-run report: %v", scanConfigID, err)
		return
	}
	if !found {
		log.Debugf("Compliance report scheduler: scan configuration %q not found, no re-run report scheduled", scanConfigID)
		return
	}
	m.scheduleOneShotReport(scanConfig)
}

// ScheduleFirstScanReport schedules a one-shot report job for the scan configuration's first
// scan cycle, due one delay after now. It is skipped when the recurring entry already reports
// the cycle in progress -- its first fire falls within the delay window, which happens when
// the configuration was created shortly after a schedule tick -- so the creation cycle is
// reported exactly once.
func (m *managerImpl) ScheduleFirstScanReport(scanConfig *storage.ComplianceOperatorScanConfigurationV2) {
	if !m.enabled() {
		return
	}
	// One-time scans have no recurring entry: the creation cycle is their only report.
	if !scanConfig.GetOneTimeScan() {
		if s, err := newReportSchedule(scanConfig); err == nil {
			if firstFire, err := s.NextFireAfter(time.Now()); err == nil &&
				!firstFire.After(time.Now().Add(env.ComplianceReportScheduleDelay.DurationSetting())) {
				// The recurring entry fires within the delay window, reporting the cycle in
				// progress -- the cycle the creation scans belong to.
				return
			}
		}
	}
	m.scheduleOneShotReport(scanConfig)
}

// scheduleOneShotReport schedules a report job due one delay after now, anchored at now. A
// pending one-shot report for the configuration is replaced, so repeated triggers (re-runs,
// creation) coalesce into one report due one delay after the latest trigger.
func (m *managerImpl) scheduleOneShotReport(scanConfig *storage.ComplianceOperatorScanConfigurationV2) {
	scheduledAt := time.Now()
	fireAt := scheduledAt.Add(env.ComplianceReportScheduleDelay.DurationSetting())
	concurrency.WithLock(&m.cronJobsLock, func() {
		if oldID, ok := m.oneShotEntryIDs[scanConfig.GetId()]; ok {
			m.cron.Remove(oldID)
			delete(m.oneShotEntryIDs, scanConfig.GetId())
		}
		entryID := m.cron.Schedule(&oneShotSchedule{fireAt: fireAt}, cron.FuncJob(func() {
			// The cycle is anchored at the trigger: scans that started since are the cycle's.
			m.submitJob(&reportJob{
				scanConfigID:   scanConfig.GetId(),
				scanConfigName: scanConfig.GetScanConfigName(),
				cycleStart:     scheduledAt,
			})
		}))
		m.oneShotEntryIDs[scanConfig.GetId()] = entryID
	})
}

// upsertScanConfigNoLock creates or replaces the cron entry of the scan configuration. The
// caller must hold cronJobsLock.
func (m *managerImpl) upsertScanConfigNoLock(scanConfig *storage.ComplianceOperatorScanConfigurationV2) {
	s, err := newReportSchedule(scanConfig)
	if err != nil {
		// Not an error: configurations without an evaluable schedule are reported on demand
		// only. Drop any entry left over from before the configuration lost its schedule.
		log.Debugf("Compliance report scheduler: no report schedule for scan configuration %q: %v", scanConfig.GetScanConfigName(), err)
		m.removeScanConfigNoLock(scanConfig.GetId())
		return
	}
	m.removeScanConfigNoLock(s.scanConfigID)
	entryID := m.cron.Schedule(s, cron.FuncJob(func() {
		// A zero cycle start makes the audit classify against the cycle that most recently
		// ticked, which is the cycle this entry fired for.
		m.submitJob(&reportJob{scanConfigID: s.scanConfigID, scanConfigName: s.scanConfigName})
	}))
	m.entryIDs[s.scanConfigID] = entryID
}

// removeScanConfigNoLock drops the cron entries of the scan configuration, both its recurring
// schedule and a pending re-run report. The caller must hold cronJobsLock.
func (m *managerImpl) removeScanConfigNoLock(scanConfigID string) {
	if oldID, ok := m.entryIDs[scanConfigID]; ok {
		m.cron.Remove(oldID)
		delete(m.entryIDs, scanConfigID)
	}
	if oldID, ok := m.oneShotEntryIDs[scanConfigID]; ok {
		m.cron.Remove(oldID)
		delete(m.oneShotEntryIDs, scanConfigID)
	}
}

// oneShotSchedule is a cron.Schedule that runs once at the given time: Next returns the fire
// time while it is still in the future and the zero time afterwards, so the entry fires at
// most once and robfig then ignores it. An entry whose fire time already passed when the cron
// processed it never runs.
type oneShotSchedule struct {
	fireAt time.Time
}

var _ cron.Schedule = (*oneShotSchedule)(nil)

// Next implements cron.Schedule.
func (o *oneShotSchedule) Next(after time.Time) time.Time {
	if o.fireAt.After(after) {
		return o.fireAt
	}
	return time.Time{}
}
