// Package settle implements the "settle, audit, snapshot" primitive for
// compliance reports: reports run on their own schedule (or on demand) instead
// of being triggered by scan completion.
//
// The completion-triggered design waits for *all* scans of a scan
// configuration across all clusters, so a partial scan cycle (node-role
// reconciliation, a sensor restart, a new MachineConfigPool, a staggered
// cluster) makes the report wait for scans that will never arrive, time out,
// and still be delivered as a success. This primitive replaces the
// unobservable question ("which scans belong to the current cycle?", which
// only sensor and the Compliance Operator can answer) with an observable one
// ("which scans are running right now?", which the scan datastore answers):
//
//   - Settle: wait, bounded, for the scans of the configuration that are
//     currently running to reach a terminal state. A scan that was running and
//     did not finish is a real failure.
//   - Audit: classify every scan of the configuration, per cluster, as fresh
//     (started within the expected schedule window), unfinished (was waited on
//     and did not finish) or overdue (missed the expected window), and derive
//     a reason for each finding, reusing the validator's taxonomy (sensor
//     disconnected, compliance operator unhealthy, scan removed, timeout).
//   - Snapshot: read the latest results per scan from the check-results
//     datastore (unchanged, see the results aggregator) and deliver, labeling
//     overdue/unfinished scans as partial failures instead of delivering a
//     success with a dangling timeout error.
//
// The manager wires this primitive into both report paths when
// ROX_COMPLIANCE_REPORTS_OWN_SCHEDULER is enabled: on-demand reports
// (manager.handleReportRequestSettled) and scheduled reports
// (manager.reportScheduler, which fires report jobs on their own schedule
// instead of on scan completion; manager.handleReadyScan no longer spawns
// report cycles from the ScanConfigWatcher).
//
// See the "Compliance reporting on partial scan cycles" proposal (proposal 1)
// and README.md in this package.
package settle

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/pkg/errors"
	complianceIntegrationDS "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	"github.com/stackrox/rox/central/complianceoperator/v2/report"
	"github.com/stackrox/rox/central/complianceoperator/v2/report/manager/watcher"
	scanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

var log = logging.LoggerForModule()

// scanPhaseDone is the Compliance Operator's terminal scan phase: the run
// finished and its results are available. The phases leading up to it
// (PENDING, LAUNCHING, RUNNING, AGGREGATING) mean the scan is in flight.
const scanPhaseDone = "DONE"

// ClusterStatusSource provides per-cluster health status. It is satisfied by
// the cluster datastore; the narrow interface keeps the settler decoupled and
// easy to fake in tests.
type ClusterStatusSource interface {
	GetCluster(ctx context.Context, id string) (*storage.Cluster, bool, error)
}

// Settler implements the settle and audit phases of the primitive. The settle
// phase polls the scan datastore -- the source of truth -- because in-memory
// watchers are a signal, not truth, and do not survive Central restarts.
type Settler struct {
	scanDS        scanDS.DataStore
	integrationDS complianceIntegrationDS.DataStore
	clusters      ClusterStatusSource

	timeout      time.Duration
	pollInterval time.Duration
	grace        time.Duration

	// now and sleep are injectable for tests.
	now   func() time.Time
	sleep func(ctx context.Context, d time.Duration) bool
}

// NewSettler returns a Settler configured from the environment settings
// (ROX_COMPLIANCE_REPORT_SETTLE_TIMEOUT, ROX_COMPLIANCE_REPORT_SETTLE_POLL_INTERVAL,
// ROX_COMPLIANCE_REPORT_SCHEDULE_GRACE).
func NewSettler(scanDataStore scanDS.DataStore, integrationDataStore complianceIntegrationDS.DataStore, clusters ClusterStatusSource) *Settler {
	return &Settler{
		scanDS:        scanDataStore,
		integrationDS: integrationDataStore,
		clusters:      clusters,
		timeout:       env.ComplianceReportSettleTimeout.DurationSetting(),
		pollInterval:  env.ComplianceReportSettlePollInterval.DurationSetting(),
		grace:         env.ComplianceReportScheduleGrace.DurationSetting(),
		now:           time.Now,
		sleep:         sleepWithContext,
	}
}

// sleepWithContext sleeps for d, returning false if the context was done first.
func sleepWithContext(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// SettleResult captures what the settle phase observed.
type SettleResult struct {
	// Running are the scans that were in flight (non-terminal phase in the scan
	// datastore) when the settle phase started.
	Running []*storage.ComplianceOperatorScanV2
	// Unfinished are the scans still in flight when the settle timeout expired.
	// These are real failures: the report waited for them and they did not
	// finish, so the report is labeled partial with the scans named.
	Unfinished []*storage.ComplianceOperatorScanV2
	// Removed are the scans that disappeared from the scan datastore while
	// being waited on.
	Removed []*storage.ComplianceOperatorScanV2
}

// AuditResult is the outcome of the expectation audit.
type AuditResult struct {
	// FailedClusters holds the per-cluster findings, keyed by cluster ID, in
	// the shape the report machinery already consumes: a non-empty map labels
	// the report PARTIAL, and a map covering every cluster of the
	// configuration labels it FAILURE.
	FailedClusters map[string]*report.FailedCluster
	// Scans records every scan covered by the report (scan ref and last start)
	// to be stored on the report snapshot, so scans already accounted for by a
	// settled report are recognized as already handled.
	Scans []*storage.ComplianceOperatorReportSnapshotV2_Scan
}

// Settle waits, bounded by ROX_COMPLIANCE_REPORT_SETTLE_TIMEOUT, for the scans
// of the scan configuration that are currently running to reach a terminal
// state. Scans that are not running are not waited on: that is the point of
// the primitive, and it is what retires the "waits for scans that will never
// arrive" failure mode of completion-triggered reporting.
func (s *Settler) Settle(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2) (*SettleResult, error) {
	scans, err := s.scansOfConfiguration(ctx, scanConfig)
	if err != nil {
		return nil, err
	}
	res := &SettleResult{}
	running := make(map[string]*storage.ComplianceOperatorScanV2)
	for _, scan := range scans {
		if scanInFlight(scan) {
			running[scanKey(scan)] = scan
		}
	}
	if len(running) == 0 {
		return res, nil
	}
	res.Running = scanMapValues(running)
	log.Infof("Settling %d running scans of scan configuration %q before generating the report (timeout: %s)",
		len(running), scanConfig.GetScanConfigName(), s.timeout)

	deadline := s.now().Add(s.timeout)
	for len(running) > 0 {
		remaining := deadline.Sub(s.now())
		if remaining <= 0 {
			res.Unfinished = scanMapValues(running)
			break
		}
		if !s.sleep(ctx, min(s.pollInterval, remaining)) {
			return nil, errors.Wrap(ctx.Err(), "settling interrupted")
		}
		scans, err := s.scansOfConfiguration(ctx, scanConfig)
		if err != nil {
			return nil, err
		}
		stillRunning := make(map[string]*storage.ComplianceOperatorScanV2)
		seen := set.NewStringSet()
		for _, scan := range scans {
			key := scanKey(scan)
			if _, waitedOn := running[key]; !waitedOn {
				continue
			}
			seen.Add(key)
			if scanInFlight(scan) {
				stillRunning[key] = scan
			}
		}
		for key, scan := range running {
			if !seen.Contains(key) {
				log.Infof("Scan %q of cluster %q disappeared while being waited on", scan.GetScanName(), scan.GetClusterId())
				res.Removed = append(res.Removed, scan)
			}
		}
		running = stillRunning
	}
	if len(res.Unfinished) > 0 {
		log.Warnf("Settle timeout (%s) expired with %d unfinished scans of scan configuration %q: %v",
			s.timeout, len(res.Unfinished), scanConfig.GetScanConfigName(), scanNames(res.Unfinished))
	}
	return res, nil
}

// Audit classifies every scan of the scan configuration, per cluster, as
// fresh, unfinished or overdue, and folds the findings into FailedCluster
// entries for the report snapshot:
//
//   - unfinished scans are listed as failed scans, so their (mid-run,
//     incoherent) results are excluded from the report content;
//   - overdue scans keep their latest results in the report (mixed freshness
//     is intended semantics) while still flagging the cluster, so the report
//     is labeled partial;
//   - clusters without a single scan of the configuration are flagged with the
//     derivable reasons (sensor disconnected, compliance operator unhealthy),
//     which is the failure visibility the wait-for-all set used to provide.
func (s *Settler) Audit(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2, settleRes *SettleResult) (*AuditResult, error) {
	scans, err := s.scansOfConfiguration(ctx, scanConfig)
	if err != nil {
		return nil, err
	}
	now := s.now()
	cutoff, hasExpectation := s.expectationCutoff(scanConfig, now)

	unfinished := set.NewStringSet()
	for _, scan := range settleRes.Unfinished {
		unfinished.Add(scanKey(scan))
	}

	ret := &AuditResult{
		FailedClusters: make(map[string]*report.FailedCluster),
	}
	scansByCluster := make(map[string][]*storage.ComplianceOperatorScanV2)
	for _, scan := range scans {
		scansByCluster[scan.GetClusterId()] = append(scansByCluster[scan.GetClusterId()], scan)
		if scan.GetLastStartedTime() != nil {
			ret.Scans = append(ret.Scans, &storage.ComplianceOperatorReportSnapshotV2_Scan{
				ScanRefId:       scan.GetScanRefId(),
				LastStartedTime: scan.GetLastStartedTime(),
			})
		}
	}
	sort.Slice(ret.Scans, func(i, j int) bool {
		return ret.Scans[i].GetScanRefId() < ret.Scans[j].GetScanRefId()
	})

	for _, cluster := range scanConfig.GetClusters() {
		clusterID := cluster.GetClusterId()
		clusterScans := scansByCluster[clusterID]
		clusterScanKeys := set.NewStringSet()
		for _, scan := range clusterScans {
			clusterScanKeys.Add(scanKey(scan))
		}
		fc := &report.FailedCluster{ClusterId: clusterID}
		for _, scan := range clusterScans {
			switch {
			case unfinished.Contains(scanKey(scan)):
				fc.FailedScans = append(fc.FailedScans, scan)
				fc.Reasons = append(fc.Reasons, fmt.Sprintf(report.SCAN_SETTLE_TIMEOUT_FMT, scan.GetScanName()))
			case !hasExpectation:
				// No cron-based expectation (one-time scan, or no scheduled fire
				// since the configuration was created): nothing to be overdue
				// against.
			case scan.GetLastStartedTime() == nil:
				fc.Reasons = append(fc.Reasons, fmt.Sprintf(report.SCAN_NEVER_STARTED_FMT, scan.GetScanName()))
			case scan.GetLastStartedTime().AsTime().Before(cutoff):
				fc.Reasons = append(fc.Reasons, fmt.Sprintf(report.SCAN_OVERDUE_FMT,
					scan.GetScanName(), scan.GetLastStartedTime().AsTime().Format(time.RFC3339)))
			}
		}
		// Scans that disappeared while being waited on are no longer in the
		// datastore, so they are flagged from the settle result rather than
		// from the scan list.
		for _, scan := range settleRes.Removed {
			if scan.GetClusterId() == clusterID && !clusterScanKeys.Contains(scanKey(scan)) {
				fc.Reasons = append(fc.Reasons, fmt.Sprintf(report.SCAN_REMOVED_FMT, scan.GetScanName()))
			}
		}
		if len(clusterScans) == 0 && hasExpectation {
			// No scan of the configuration exists for this cluster at all:
			// nothing has ever arrived (e.g. the sensor has been down since
			// before the last expected run). Today the machinery is blind to
			// this case because it never starts; the audit flags it.
			fc.Reasons = append(fc.Reasons, report.NO_SCANS_OF_CONFIG_FOUND)
		}
		if len(fc.Reasons) == 0 {
			continue
		}
		// Diagnostics for the findings: sensor connectivity and compliance
		// operator health, i.e. the taxonomy the scan validators already
		// produce, relocated from wait-timeout time to report time.
		fc.Reasons = append(fc.Reasons, s.diagnostics(ctx, clusterID)...)
		// Overdue-only findings keep the cluster's (possibly stale) results in
		// the report; only unfinished scans are excluded from the content.
		fc.IncludeResults = len(fc.FailedScans) == 0 && len(clusterScans) > 0
		log.Infof("Scan configuration %q: cluster %q flagged with reasons %v", scanConfig.GetScanConfigName(), clusterID, fc.Reasons)
		ret.FailedClusters[clusterID] = fc
	}
	return ret, nil
}

// diagnostics derives per-cluster reasons for audit findings: sensor
// connectivity from the cluster status and compliance operator health from the
// compliance integration, the same taxonomy ValidateClusterHealth produces.
func (s *Settler) diagnostics(ctx context.Context, clusterID string) []string {
	var reasons []string
	if s.clusters != nil {
		if cluster, found, err := s.clusters.GetCluster(ctx, clusterID); err == nil && found {
			if cluster.GetHealthStatus().GetSensorHealthStatus() != storage.ClusterHealthStatus_HEALTHY {
				reasons = append(reasons, report.SENSOR_DISCONNECTED)
			}
		}
	}
	if _, err := watcher.IsComplianceOperatorHealthy(ctx, clusterID, s.integrationDS); err != nil {
		switch {
		case errors.Is(err, watcher.ErrComplianceOperatorNotInstalled):
			reasons = append(reasons, report.COMPLIANCE_NOT_INSTALLED)
		case errors.Is(err, watcher.ErrComplianceOperatorVersion):
			reasons = append(reasons, report.COMPLIANCE_VERSION_ERROR)
		}
	}
	return reasons
}

// scansOfConfiguration returns the scans of the scan configuration across all
// of its clusters, from the scan datastore. Scans carry the suite label
// matching the scan configuration name (see
// internaltov2storage.ComplianceOperatorScanObject); this is the same query
// the report cluster-data helper uses.
func (s *Settler) scansOfConfiguration(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2) ([]*storage.ComplianceOperatorScanV2, error) {
	clusterIDs := make([]string, 0, len(scanConfig.GetClusters()))
	for _, cluster := range scanConfig.GetClusters() {
		clusterIDs = append(clusterIDs, cluster.GetClusterId())
	}
	if len(clusterIDs) == 0 {
		return nil, nil
	}
	query := search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanConfigName, scanConfig.GetScanConfigName()).
		AddExactMatches(search.ClusterID, clusterIDs...).
		ProtoQuery()
	scans, err := s.scanDS.SearchScans(ctx, query)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to retrieve scans of scan configuration %q", scanConfig.GetScanConfigName())
	}
	return scans, nil
}

// scanInFlight reports whether the scan's run has not finished yet. The
// Compliance Operator transitions scans PENDING -> LAUNCHING -> RUNNING ->
// AGGREGATING -> DONE and leaves them at DONE (with a result, possibly ERROR)
// between runs. A scan with no observed phase is not waited on: there is
// nothing observable to wait for, and the audit classifies it by its start
// time instead.
func scanInFlight(scan *storage.ComplianceOperatorScanV2) bool {
	phase := scan.GetStatus().GetPhase()
	return phase != "" && phase != scanPhaseDone
}

// scanKey identifies a scan watcher-style, by cluster and scan resource ID.
func scanKey(scan *storage.ComplianceOperatorScanV2) string {
	return fmt.Sprintf("%s:%s", scan.GetClusterId(), scan.GetId())
}

func scanMapValues(m map[string]*storage.ComplianceOperatorScanV2) []*storage.ComplianceOperatorScanV2 {
	ret := scanMapValuesUnsorted(m)
	sort.Slice(ret, func(i, j int) bool {
		return scanKey(ret[i]) < scanKey(ret[j])
	})
	return ret
}

func scanMapValuesUnsorted(m map[string]*storage.ComplianceOperatorScanV2) []*storage.ComplianceOperatorScanV2 {
	ret := make([]*storage.ComplianceOperatorScanV2, 0, len(m))
	for _, scan := range m {
		ret = append(ret, scan)
	}
	return ret
}

func scanNames(scans []*storage.ComplianceOperatorScanV2) []string {
	ret := make([]string, 0, len(scans))
	for _, scan := range scans {
		ret = append(ret, scan.GetScanName())
	}
	return ret
}
