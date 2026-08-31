package helpers

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	checkResultDS "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	"github.com/stackrox/rox/central/complianceoperator/v2/report"
	snapshotDS "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore"
	scanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/search"
	"google.golang.org/protobuf/types/known/timestamppb"
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

// DetectStaleClusters identifies clusters whose check results are older than
// the scan configuration's last execution time. This happens when a scan was
// triggered but results never arrived (e.g. sensor disconnect, watcher timeout).
// Without this check, the on-demand report path silently serves stale data from
// a previous scan cycle.
func DetectStaleClusters(
	ctx context.Context,
	reportData *storage.ComplianceOperatorReportData,
	existingFailedClusters map[string]*report.FailedCluster,
	checkResultStore checkResultDS.DataStore,
) map[string]*report.FailedCluster {
	lastExec := reportData.GetLastExecutedTime()
	if lastExec == nil {
		// No execution recorded yet — nothing to compare against.
		return nil
	}

	staleClusters := make(map[string]*report.FailedCluster)
	scanConfigID := reportData.GetScanConfiguration().GetId()

	for _, cluster := range reportData.GetClusterStatus() {
		clusterID := cluster.GetClusterId()
		if _, alreadyFailed := existingFailedClusters[clusterID]; alreadyFailed {
			continue
		}

		latestResultTime := getLatestCheckResultTime(ctx, scanConfigID, clusterID, checkResultStore)
		if latestResultTime == nil || protocompat.CompareTimestamps(latestResultTime, lastExec) < 0 {
			var assessmentStr string
			if latestResultTime == nil {
				assessmentStr = "none"
			} else {
				assessmentStr = latestResultTime.AsTime().UTC().Format(time.RFC1123)
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
	}
	return staleClusters
}

// getLatestCheckResultTime returns the most recent LastStartedTime among
// check results for the given scan config and cluster, or nil if none exist.
func getLatestCheckResultTime(
	ctx context.Context,
	scanConfigID, clusterID string,
	checkResultStore checkResultDS.DataStore,
) *timestamppb.Timestamp {
	// Query one result sorted by LastStartedTime DESC.
	q := search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanConfig, scanConfigID).
		AddExactMatches(search.ClusterID, clusterID).
		WithPagination(
			search.NewPagination().
				Limit(1).
				AddSortOption(search.NewSortOption(search.ComplianceOperatorCheckLastStartedTime).Reversed(true)),
		).
		ProtoQuery()
	results, err := checkResultStore.SearchComplianceCheckResults(ctx, q)
	if err != nil {
		log.Warnf("Failed to query check results for cluster %s: %v", clusterID, err)
		return nil
	}
	if len(results) == 0 {
		return nil
	}
	return results[0].GetLastStartedTime()
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
