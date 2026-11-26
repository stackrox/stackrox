package scheduler

import (
	"context"
	"strings"
	"time"

	"github.com/stackrox/rox/central/complianceoperator/v2/freshness"
	"github.com/stackrox/rox/central/complianceoperator/v2/report/manager/watcher"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/search"
)

// settlePollInterval is how often the settler re-checks the running scans while waiting. It
// only bounds detection latency: a scan that finished is seen at the next poll.
const settlePollInterval = 30 * time.Second

// resolveCycleStart returns the anchor of the scan cycle the report belongs to: the explicit
// anchor carried by the job (re-run and first-scan reports), or, for scheduled fires, the
// schedule tick that most recently passed. It returns false when no anchor can be derived,
// which happens when the configuration has no evaluable schedule.
//
// The anchor is resolved once per report job so that the settler and the expectation audit
// classify against the same cycle even if a schedule tick passes while settling.
func resolveCycleStart(scanConfig *storage.ComplianceOperatorScanConfigurationV2, cycleStart time.Time) (time.Time, bool) {
	if !cycleStart.IsZero() {
		return cycleStart, true
	}
	return freshness.PreviousScheduledStart(scanConfig.GetSchedule(), time.Now().UTC())
}

// settleRunningScans is the settler of the decoupled reporting redesign: the report schedule
// delay is a heuristic, so the cycle's scans may still be running when the report fires. It
// waits, bounded by ROX_COMPLIANCE_REPORT_SETTLE_TIMEOUT, for them to finish, so the
// expectation audit classifies them as fresh instead of unfinished. Scans still running when
// the bound expires are reported as unfinished: an honest PARTIAL, never a silent hole.
//
// Only scans of the current cycle are waited for: a scan whose last start predates the cycle
// is a failure either way (unfinished or overdue), and waiting for it would only delay the
// report.
func (m *managerImpl) settleRunningScans(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2, cycleStart time.Time) {
	cutoff := cycleStart.Add(-auditFreshnessTolerance)
	deadline := time.Now().Add(env.ComplianceReportSettleTimeout.DurationSetting())
	for {
		running, err := m.runningScansOfCycle(ctx, scanConfig, cutoff)
		if err != nil {
			// A read failure must not skip the report: the audit classifies what it can see.
			log.Warnf("Compliance report scheduler: unable to retrieve the running scans of scan configuration %q: %v", scanConfig.GetScanConfigName(), err)
			return
		}
		if len(running) == 0 {
			return
		}
		if !time.Now().Before(deadline) {
			names := make([]string, 0, len(running))
			for _, scan := range running {
				names = append(names, scan.GetScanName())
			}
			log.Infof("Compliance report scheduler: the settle window of scan configuration %q expired waiting for %d scan(s) still running: %s",
				scanConfig.GetScanConfigName(), len(running), strings.Join(names, ", "))
			return
		}
		log.Debugf("Compliance report scheduler: waiting for %d scan(s) of scan configuration %q to finish", len(running), scanConfig.GetScanConfigName())
		select {
		case <-ctx.Done():
			// Shutdown: stop waiting. Generation proceeds and fails on the cancelled context.
			return
		case <-time.After(min(settlePollInterval, time.Until(deadline))):
		}
	}
}

// runningScansOfCycle returns the scans of the scan configuration that are still running and
// belong to the current cycle: not finished, not not-applicable, and last started at or after
// the cutoff. The datastore is the source of truth; the in-memory scan watchers are a signal,
// not truth, and do not survive Central restarts.
func (m *managerImpl) runningScansOfCycle(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2, cutoff time.Time) ([]*storage.ComplianceOperatorScanV2, error) {
	scans, err := m.scanDataStore.SearchScans(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanConfigName, scanConfig.GetScanConfigName()).
		ProtoQuery())
	if err != nil {
		return nil, err
	}
	var running []*storage.ComplianceOperatorScanV2
	for _, scan := range scans {
		if scan.GetStatus().GetPhase() == scanPhaseDone || watcher.IsScanNotApplicable(scan) {
			continue
		}
		if scan.GetLastStartedTime() == nil || scan.GetLastStartedTime().AsTime().Before(cutoff) {
			continue
		}
		running = append(running, scan)
	}
	return running, nil
}
