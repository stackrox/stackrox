package manager

import (
	"context"
	"time"

	"github.com/pkg/errors"
	snapshotDS "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore"
	"github.com/stackrox/rox/central/complianceoperator/v2/report/manager/settle"
	scanConfigurationDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/search"
)

// errScanConfigGone is reported by a config scheduler that observed its scan
// configuration disappear from the datastore (a delete event the manager
// missed). The manager deregisters the scheduler instead of respawning it.
var errScanConfigGone = errors.New("scan configuration no longer exists")

// configScheduler runs the report schedule of a single scan configuration:
// it sleeps until the report for the current schedule fire is due (fire +
// delay) and submits it through the settle-audit-snapshot path. The
// configuration is re-read from the datastore on every wake-up, so cron
// edits, notifier changes and cluster changes are picked up without
// restarting the scheduler.
//
// The loop is self-healing: transient datastore errors and failed submissions
// are retried with backoff, and only a stop request or the disappearance of
// the configuration exits it. Unexpected exits (bugs) are still covered by
// the manager's respawn watcher (see reportSchedulerManager.watch).
type configScheduler struct {
	id  string
	ctx context.Context

	scanConfigDS scanConfigurationDS.DataStore
	snapshotDS   snapshotDS.DataStore
	// submit queues a scheduled report request for execution.
	submit func(scanConfig *storage.ComplianceOperatorScanConfigurationV2) error

	delay            time.Duration // fire -> report offset
	recheckInterval  time.Duration // idle cadence when nothing is due yet
	retryInterval    time.Duration // backoff on transient errors and failed submits
	missingThreshold int           // consecutive not-found reads before exiting

	// now and sleep are injectable for tests. sleep returns false when the
	// stop channel closed first.
	now   func() time.Time
	sleep func(stop <-chan struct{}, d time.Duration) bool

	stopper concurrency.Stopper
	// lastSubmittedFire is the in-memory half of the one-report-per-fire
	// dedup (the snapshot store covers previous processes). Only written by
	// the run loop; read by the run loop only, after Run started.
	lastSubmittedFire time.Time
}

// sleepUntilStop sleeps for d, returning false if the stop channel closed first.
func sleepUntilStop(stop <-chan struct{}, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-stop:
		return false
	case <-timer.C:
		return true
	}
}

// Run runs the scheduler loop until stopped. It reports stopped through its
// stopper, carrying errScanConfigGone when it exited because the scan
// configuration disappeared.
func (s *configScheduler) Run() {
	flow := s.stopper.Flow()
	defer flow.ReportStopped()
	missing := 0
	for {
		scanConfig, found, err := s.scanConfigDS.GetScanConfiguration(s.ctx, s.id)
		if err != nil {
			schedulerLog.Warnf("Report scheduler for %q: unable to read the scan configuration: %v", s.id, err)
			if !s.sleep(flow.StopRequested(), s.retryInterval) {
				return
			}
			continue
		}
		if !found {
			missing++
			if missing >= s.missingThreshold {
				// The configuration is gone and no delete event reached the
				// manager. Exit with the typed error so the manager
				// deregisters instead of respawning.
				schedulerLog.Infof("Report scheduler for %q: the scan configuration disappeared from the datastore", s.id)
				flow.StopWithError(errScanConfigGone)
				return
			}
			if !s.sleep(flow.StopRequested(), s.retryInterval) {
				return
			}
			continue
		}
		missing = 0

		if len(scanConfig.GetNotifiers()) == 0 {
			// Nowhere to deliver a scheduled report; recheck later in case
			// notifiers are added.
			if !s.sleep(flow.StopRequested(), s.recheckInterval) {
				return
			}
			continue
		}

		fire, due, ok := s.reportDue(scanConfig)
		if !ok || s.now().Before(due) {
			// Nothing due yet: sleep until the report for the current fire
			// is due, or recheck periodically when there is no fire to wait
			// for (e.g. the configuration is younger than its cron).
			wait := s.recheckInterval
			if ok {
				wait = due.Sub(s.now())
			}
			if !s.sleep(flow.StopRequested(), wait) {
				return
			}
			continue
		}
		if s.alreadyReported(fire) {
			// The report for this fire is accounted for; idle until the next
			// fire (the periodic wake also picks up cron edits).
			if !s.sleep(flow.StopRequested(), s.recheckInterval) {
				return
			}
			continue
		}
		if err := s.submit(scanConfig); err != nil {
			// The fire is not recorded, so it is retried (e.g. while an
			// on-demand report for the same configuration is in flight).
			schedulerLog.Warnf("Report scheduler for %q: unable to submit the scheduled report (will retry): %v", s.id, err)
			if !s.sleep(flow.StopRequested(), s.retryInterval) {
				return
			}
			continue
		}
		s.lastSubmittedFire = fire
		scheduledReportsSubmitted.WithLabelValues(scanConfig.GetScanConfigName()).Inc()
		schedulerLog.Infof("Report scheduler for %q: submitted the scheduled report (fire: %s)",
			scanConfig.GetScanConfigName(), fire.Format(time.RFC3339))
	}
}

// reportDue returns the fire the next scheduled report corresponds to and
// when it is due. Scheduled configurations report per cron fire (see
// settle.ReportDue). One-time configurations have no cron; their single
// implicit fire is the creation time, so a one-time scan still gets exactly
// one scheduled report, one delay after the configuration was created.
func (s *configScheduler) reportDue(scanConfig *storage.ComplianceOperatorScanConfigurationV2) (fire, due time.Time, ok bool) {
	if fire, due, ok := settle.ReportDue(scanConfig, s.now(), s.delay); ok {
		return fire, due, true
	}
	created := scanConfig.GetCreatedTime()
	if scanConfig.GetOneTimeScan() && created != nil {
		return created.AsTime(), created.AsTime().Add(s.delay), true
	}
	return time.Time{}, time.Time{}, false
}

// alreadyReported returns whether a scheduled report already accounts for the
// given fire: in memory (this scheduler) or in the snapshot store (previous
// processes or a respawned scheduler). A scheduled report generated at or
// after the fire covers it: the delay is expected to stay well below the scan
// interval, so the report of an earlier fire cannot leak into this fire's
// window. A failed attempt is also recorded (as a FAILURE snapshot), so there
// is at most one scheduled report attempt per fire.
func (s *configScheduler) alreadyReported(fire time.Time) bool {
	if !s.lastSubmittedFire.Before(fire) {
		return true
	}
	lastScheduled, err := s.lastScheduledSnapshot()
	if err != nil {
		// Fail open: prefer a possible duplicate report over a silently
		// missed one.
		schedulerLog.Warnf("Report scheduler for %q: unable to retrieve the last scheduled report: %v", s.id, err)
		return false
	}
	if started := lastScheduled.GetReportStatus().GetStartedAt(); started != nil && !started.AsTime().Before(fire) {
		s.lastSubmittedFire = fire
		return true
	}
	return false
}

// lastScheduledSnapshot returns the most recent SCHEDULED report snapshot of
// the scan configuration. On-demand snapshots do not count: a user-triggered
// report does not discharge the scheduled one.
func (s *configScheduler) lastScheduledSnapshot() (*storage.ComplianceOperatorReportSnapshotV2, error) {
	query := search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanConfig, s.id).
		AddExactMatches(search.ComplianceOperatorReportRequestType, storage.ComplianceOperatorReportStatus_SCHEDULED.String()).
		ProtoQuery()
	snapshots, err := s.snapshotDS.SearchSnapshots(s.ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "unable to search scheduled snapshots")
	}
	var last *storage.ComplianceOperatorReportSnapshotV2
	for _, snapshot := range snapshots {
		// The query already filters by request type; re-check in memory so
		// the "only SCHEDULED counts" semantics hold regardless of the index.
		if snapshot.GetReportStatus().GetReportRequestType() != storage.ComplianceOperatorReportStatus_SCHEDULED {
			continue
		}
		if protocompat.CompareTimestamps(snapshot.GetReportStatus().GetStartedAt(), last.GetReportStatus().GetStartedAt()) > 0 {
			last = snapshot
		}
	}
	return last, nil
}
