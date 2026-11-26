package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/stackrox/rox/central/complianceoperator/v2/freshness"
	"github.com/stackrox/rox/central/complianceoperator/v2/report"
	"github.com/stackrox/rox/central/complianceoperator/v2/report/manager/watcher"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

const (
	// scanPhaseDone is the compliance operator's terminal scan phase; scans in any other phase
	// (PENDING, LAUNCHING, RUNNING, AGGREGATING) are still in flight. The phase is transported
	// as a raw string from the operator's ComplianceScanStatusPhase.
	scanPhaseDone = "DONE"

	// auditFreshnessTolerance absorbs compliance operator trigger jitter and clock skew between
	// the cluster and Central when classifying a scan's start time against the cycle cutoff.
	// The report schedule delay is the primary buffer: by the time the report fires, the
	// cycle's scans have long since started.
	auditFreshnessTolerance = 15 * time.Minute
)

// auditExpectations is the decoupled reporting redesign's failure accounting (see
// compliance-report-partial-cycles.md, proposal 1): it replaces the report manager's
// wait-set-timeout accounting with an explicit, datastore-driven one. Every scan of the scan
// configuration, per cluster, is classified against the expected scan cycle:
//
//   - fresh: started within the current cycle and finished
//   - unfinished: still running at report time (the settler, a later iteration, waits for
//     these before the report is generated; until then they are reported as partial failures)
//   - overdue: has not started within the current cycle
//
// Clusters of the configuration without any scans at all are overdue as a whole, which keeps
// the "all sensors down" failure mode visible. The output feeds the existing labeling: a
// non-empty map makes the report PARTIAL (or FAILURE when every cluster fails), never
// DELIVERED with a dangling error.
//
// cycleStart anchors the current cycle: re-run reports pass the re-run time; a zero value
// means "the cycle that most recently ticked", derived from the configuration's schedule.
func (m *managerImpl) auditExpectations(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2, cycleStart time.Time) map[string]*report.FailedCluster {
	clusters := scanConfig.GetClusters()
	if len(clusters) == 0 {
		return nil
	}

	cutoff := cycleStart
	if cutoff.IsZero() {
		var ok bool
		cutoff, ok = freshness.PreviousScheduledStart(scanConfig.GetSchedule(), time.Now().UTC())
		if !ok {
			// No evaluable schedule (e.g. the configuration lost it after the entry fired):
			// no accounting rather than a wrong one.
			log.Debugf("Compliance report scheduler: no evaluable schedule for scan configuration %q, skipping the expectation audit", scanConfig.GetScanConfigName())
			return nil
		}
	}
	cutoff = cutoff.Add(-auditFreshnessTolerance)

	// The scan universe is the same one the report content is assembled from: every scan
	// referencing the configuration, across its clusters.
	scans, err := m.scanDataStore.SearchScans(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanConfigName, scanConfig.GetScanConfigName()).
		ProtoQuery())
	if err != nil {
		log.Errorf("Compliance report scheduler: unable to retrieve the scans of scan configuration %q for the expectation audit: %v", scanConfig.GetScanConfigName(), err)
		return nil
	}
	scansByCluster := make(map[string][]*storage.ComplianceOperatorScanV2, len(clusters))
	for _, scan := range scans {
		scansByCluster[scan.GetClusterId()] = append(scansByCluster[scan.GetClusterId()], scan)
	}

	failedClusters := make(map[string]*report.FailedCluster)
	for _, cluster := range clusters {
		clusterID := cluster.GetClusterId()
		clusterScans := scansByCluster[clusterID]

		var failedScans []*storage.ComplianceOperatorScanV2
		var reasons []string
		applicableScanSeen := false
		for _, scan := range clusterScans {
			if watcher.IsScanNotApplicable(scan) {
				continue
			}
			applicableScanSeen = true
			switch {
			case scan.GetStatus().GetPhase() != scanPhaseDone:
				failedScans = append(failedScans, scan)
				reasons = append(reasons, fmt.Sprintf(report.SCAN_STILL_RUNNING_FMT, scan.GetScanName()))
			case scan.GetLastStartedTime() == nil || scan.GetLastStartedTime().AsTime().Before(cutoff):
				failedScans = append(failedScans, scan)
				reasons = append(reasons, fmt.Sprintf(report.SCAN_NOT_STARTED_FMT, scan.GetScanName()))
			}
		}

		switch {
		case len(clusterScans) == 0:
			// The cluster has no scans for this configuration at all.
			failedClusters[clusterID] = m.failedCluster(ctx, clusterID, nil, []string{report.NO_SCANS_STARTED})
		case !applicableScanSeen:
			// Every scan of the cluster is NOT-APPLICABLE: reported like the report manager
			// reports it, as a cluster-level note rather than a scan failure.
			failedClusters[clusterID] = &report.FailedCluster{
				ClusterId: clusterID,
				Reasons:   []string{report.SCAN_NOT_APPLICABLE},
			}
		case len(failedScans) > 0:
			failedClusters[clusterID] = m.failedCluster(ctx, clusterID, failedScans, reasons)
		}
	}
	return failedClusters
}

// failedCluster assembles the failed-cluster entry for the cluster. Compliance operator health
// reasons take precedence over the scan-level reasons, mirroring the report manager's
// validator: an installation or operator problem explains the whole cluster.
func (m *managerImpl) failedCluster(ctx context.Context, clusterID string, failedScans []*storage.ComplianceOperatorScanV2, reasons []string) *report.FailedCluster {
	health := watcher.ValidateClusterHealth(ctx, clusterID, m.integrationDataStore)
	if len(health.Reasons) > 0 {
		return &report.FailedCluster{
			ClusterId:       clusterID,
			Reasons:         health.Reasons,
			OperatorVersion: health.OperatorVersion,
			FailedScans:     failedScans,
		}
	}
	return &report.FailedCluster{
		ClusterId:   clusterID,
		Reasons:     reasons,
		FailedScans: failedScans,
	}
}
