package helpers

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/complianceoperator/v2/report"
	snapshotDS "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore"
	scanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/search"
)

// GetFailedClusters returns the failed clusters metadata associated with a ScanConfiguration
func GetFailedClusters(ctx context.Context, scanConfigID string, snapshotStore snapshotDS.DataStore, scanStore scanDS.DataStore) (map[string]*report.FailedCluster, error) {
	failedClusters := make(map[string]*report.FailedCluster)
	prevSnapshot, err := snapshotStore.GetLastSnapshotFromScanConfig(ctx, scanConfigID)
	if err != nil {
		return nil, err
	}
	for _, failedCluster := range prevSnapshot.GetFailedClusters() {
		scans, err := populateFailedScans(ctx, failedCluster.GetScanNames(), prevSnapshot.GetScans(), scanStore)
		if err != nil {
			return nil, err
		}
		failedClusters[failedCluster.GetClusterId()] = &report.FailedCluster{
			ClusterId:       failedCluster.GetClusterId(),
			ClusterName:     failedCluster.GetClusterName(),
			Reasons:         failedCluster.GetReasons(),
			OperatorVersion: failedCluster.GetOperatorVersion(),
			FailedScans:     scans,
		}
	}
	return failedClusters, nil
}

// GetClusterData returns the cluster metadata associated with a report data
func GetClusterData(ctx context.Context, reportData *storage.ComplianceOperatorReportData, failedClusters map[string]*report.FailedCluster, scanStore scanDS.DataStore) (map[string]*report.ClusterData, error) {
	clusterData := make(map[string]*report.ClusterData)
	for _, cluster := range reportData.GetClusterStatus() {
		data := &report.ClusterData{
			ClusterId:   cluster.GetClusterId(),
			ClusterName: cluster.GetClusterName(),
		}
		data, err := populateScanNames(ctx, data, reportData, cluster.GetClusterId(), scanStore)
		if err != nil {
			return nil, err
		}
		clusterData[cluster.GetClusterId()] = data
	}
	for failedClusterId, failedCluster := range failedClusters {
		cluster, found := clusterData[failedClusterId]
		if !found {
			continue
		}

		failedCluster.ClusterName = cluster.ClusterName
		cluster.FailedInfo = failedCluster
	}
	return clusterData, nil
}

func populateScanNames(ctx context.Context, data *report.ClusterData, reportData *storage.ComplianceOperatorReportData, clusterID string, scanStore scanDS.DataStore) (*report.ClusterData, error) {
	if data == nil {
		return nil, errors.New("cannot populate scans and profiles of a nil ClusterData")
	}
	query := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, clusterID).
		AddExactMatches(search.ComplianceOperatorScanConfigName, reportData.GetScanConfiguration().GetScanConfigName()).
		ProtoQuery()
	scans, err := scanStore.SearchScans(ctx, query)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to retrieve scans associated with the ScanConfiguration %q in the cluster %q", reportData.GetScanConfiguration().GetId(), clusterID)
	}
	for _, scan := range scans {
		data.ScanNames = append(data.ScanNames, scan.GetScanName())
	}
	return data, nil
}

var log = logging.LoggerForModule()

// DetectStaleClusters identifies clusters that did not complete the scan
// configuration's latest execution: their suite's last transition time is older
// than the configuration's overall last execution time. This happens when a
// scan was triggered but a cluster never produced fresh results (e.g. sensor
// disconnect, watcher timeout in a previous cycle). Without this check, the
// on-demand report path silently serves stale data from a previous scan cycle,
// unlike the scheduled path which flags such clusters via the ScanConfigWatcher.
//
// The comparison is like-with-like: both the per-cluster suite time and the
// config's LastExecutedTime are suite transition timestamps, so a cluster that
// participated in the latest run compares equal (or newer) and is not flagged.
func DetectStaleClusters(
	reportData *storage.ComplianceOperatorReportData,
	existingFailedClusters map[string]*report.FailedCluster,
) map[string]*report.FailedCluster {
	lastExec := reportData.GetLastExecutedTime()
	if lastExec == nil {
		// No execution recorded yet — nothing to compare against.
		return nil
	}

	staleClusters := make(map[string]*report.FailedCluster)
	for _, cluster := range reportData.GetClusterStatus() {
		clusterID := cluster.GetClusterId()
		if _, alreadyFailed := existingFailedClusters[clusterID]; alreadyFailed {
			continue
		}

		clusterTime := cluster.GetSuiteStatus().GetLastTransitionTime()
		if clusterTime != nil && protocompat.CompareTimestamps(clusterTime, lastExec) >= 0 {
			// Cluster completed the latest execution; not stale.
			continue
		}

		assessmentStr := "none"
		if clusterTime != nil {
			assessmentStr = clusterTime.AsTime().UTC().Format(time.RFC1123)
		}
		reason := fmt.Sprintf(report.SCAN_RESULTS_STALE_FMT,
			assessmentStr,
			lastExec.AsTime().UTC().Format(time.RFC1123),
		)
		log.Warnf("Cluster %s (%s) has stale scan results: %s",
			clusterID, cluster.GetClusterName(), reason)
		staleClusters[clusterID] = &report.FailedCluster{
			ClusterId:   clusterID,
			ClusterName: cluster.GetClusterName(),
			Reasons:     []string{reason},
		}
	}
	return staleClusters
}

func populateFailedScans(ctx context.Context, failedScanNames []string, snapshotScans []*storage.ComplianceOperatorReportSnapshotV2_Scan, scanStore scanDS.DataStore) ([]*storage.ComplianceOperatorScanV2, error) {
	scanRefIDs := make([]string, 0, len(snapshotScans))
	for _, scan := range snapshotScans {
		scanRefIDs = append(scanRefIDs, scan.GetScanRefId())
	}
	// We need to query by ScanName and ScanRefIDs
	// because ScanNames are not unique cross cluster.
	// scanRefIDs holds all the scan references (failed and successful)
	// associated with the ScanConfiguration.
	// failedScanNames holds the scan names of the failed scans.
	query := search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanName, failedScanNames...).
		AddExactMatches(search.ComplianceOperatorScanResult, scanRefIDs...).ProtoQuery()
	scans, err := scanStore.SearchScans(ctx, query)
	if err != nil {
		return nil, err
	}
	return scans, nil
}
