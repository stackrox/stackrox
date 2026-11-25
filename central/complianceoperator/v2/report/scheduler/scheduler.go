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
// re-run-triggered reports) will submit jobs through the same queue without touching the cron.
//
// Waiting for scan completion is replaced by settling currently-running scans at report time
// and auditing schedule expectations against the datastore; those steps land in later
// iterations of the rollout, so for now the fired job is a no-op.
package scheduler

import (
	"context"
	"sync/atomic"

	scanConfigDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
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
type Manager interface {
	// Start starts the scheduler: it reconciles one cron entry per stored scan configuration
	// with an evaluable schedule and then runs submitted report jobs. Start is idempotent.
	Start()
	// Stop stops the scheduler and releases its cron entries. Stop is idempotent; a stopped
	// scheduler cannot be restarted.
	Stop()
	// UpsertScanConfig creates or updates the cron entry of the given scan configuration,
	// e.g. on configuration create or update. Configurations without an evaluable schedule
	// (one-time scans) get no entry; upserting one removes any entry the configuration had.
	UpsertScanConfig(scanConfig *storage.ComplianceOperatorScanConfigurationV2)
	// RemoveScanConfig drops the cron entry of the scan configuration, if any, e.g. on
	// configuration deletion.
	RemoveScanConfig(scanConfigID string)
}

type managerImpl struct {
	scanConfigDS scanConfigDS.DataStore

	stopper   concurrency.Stopper
	isStarted atomic.Bool
	isStopped atomic.Bool

	// stopCtx is cancelled once shutdown is requested; it unblocks the execution loop's
	// concurrency acquisition.
	stopCtx context.Context

	// schedulerCtx scopes the scheduler's background work: no user identity, Compliance access
	// only (as the freshness sweeper). It is cancelled once shutdown is requested, so neither
	// datastore reads nor running jobs outlive Stop(). Set by Start.
	schedulerCtx context.Context

	// cron triggers one entry per scan configuration; entries submit jobs rather than running
	// them, so the cron goroutine never blocks on report generation.
	cron *cron.Cron
	// cronJobsLock guards entryIDs.
	cronJobsLock sync.Mutex
	// entryIDs maps scan configuration IDs to their cron entries.
	entryIDs map[string]cron.EntryID

	// jobQueue holds submitted report jobs; readyForReports wakes the execution loop when a
	// job is enqueued or a running one finishes.
	jobQueue        *jobQueue
	readyForReports concurrency.Signal
	concurrencySem  *semaphore.Weighted

	// runJob is the report job executed for submitted jobs; injectable for tests.
	runJob func(ctx context.Context, job *reportJob)
}

// New returns a report scheduler backed by the given scan configuration datastore.
func New(scanConfigDS scanConfigDS.DataStore) Manager {
	stopper := concurrency.NewStopper()
	return &managerImpl{
		scanConfigDS:    scanConfigDS,
		stopper:         stopper,
		stopCtx:         concurrency.AsContext(stopper.LowLevel().GetStopRequestSignal()),
		cron:            cron.New(),
		entryIDs:        make(map[string]cron.EntryID),
		jobQueue:        newJobQueue(),
		readyForReports: concurrency.NewSignal(),
		concurrencySem:  semaphore.NewWeighted(int64(env.ReportExecutionMaxConcurrency.IntegerSetting())),
		runJob:          runReportJob,
	}
}

func (m *managerImpl) Start() {
	if m.isStopped.Load() {
		log.Error("Compliance report scheduler already stopped. It cannot be re-started once stopped.")
		return
	}
	if !m.isStarted.CompareAndSwap(false, true) {
		log.Error("Compliance report scheduler already running")
		return
	}
	log.Info("Starting compliance report scheduler")

	m.schedulerCtx = sac.WithGlobalAccessScopeChecker(
		m.stopCtx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance),
		),
	)

	// Reconcile before the cron starts running so stored configurations get entries without
	// producing events (Central restarts produce none). Entries added while the cron is not
	// running are parked and scheduled when it starts.
	m.reconcile(m.schedulerCtx)
	m.cron.Start()
	go m.runJobs()
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
	})
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

	m.runJob(m.schedulerCtx, job)
}

// submitJob enqueues a report job for the scan configuration and wakes the execution loop.
// Cron entries submit here; later iterations' re-run and on-demand triggers will too.
func (m *managerImpl) submitJob(scanConfigID, scanConfigName string) {
	m.jobQueue.enqueue(&reportJob{
		scanConfigID:   scanConfigID,
		scanConfigName: scanConfigName,
	})
	m.readyForReports.Signal()
}

// reconcile rebuilds the entry set from the stored scan configurations so that exactly the
// configurations with an evaluable schedule have a cron entry.
func (m *managerImpl) reconcile(ctx context.Context) {
	configs, err := m.scanConfigDS.GetScanConfigurations(ctx, search.EmptyQuery())
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
	concurrency.WithLock(&m.cronJobsLock, func() {
		m.upsertScanConfigNoLock(scanConfig)
	})
}

func (m *managerImpl) RemoveScanConfig(scanConfigID string) {
	concurrency.WithLock(&m.cronJobsLock, func() {
		m.removeScanConfigNoLock(scanConfigID)
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
		m.submitJob(s.scanConfigID, s.scanConfigName)
	}))
	m.entryIDs[s.scanConfigID] = entryID
}

// removeScanConfigNoLock drops the cron entry of the scan configuration, if any. The caller
// must hold cronJobsLock.
func (m *managerImpl) removeScanConfigNoLock(scanConfigID string) {
	if oldID, ok := m.entryIDs[scanConfigID]; ok {
		m.cron.Remove(oldID)
		delete(m.entryIDs, scanConfigID)
	}
}
