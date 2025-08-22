package helpers

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/complianceoperator/v2/report"
	snapshotDS "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore"
	scanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	"github.com/stackrox/rox/generated/storage"
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
