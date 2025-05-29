package helpers

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/complianceoperator/v2/report"
	snapshotDS "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore"
	scanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"golang.org/x/exp/maps"
)

// GetFailedClusters returns the failed clusters metadata associated with a ScanConfiguration
func GetFailedClusters(ctx context.Context, scanConfigID string, snapshotStore snapshotDS.DataStore, scanStore scanDS.DataStore) (map[string]*report.FailedCluster, error) {
	failedClusters := make(map[string]*report.FailedCluster)
	prevSnapshot, err := snapshotStore.GetLastSnapshotFromScanConfig(ctx, scanConfigID)
	if err != nil {
		return nil, err
	}
	for _, failedCluster := range prevSnapshot.GetFailedClusters() {
		scans, err := populateFailedScans(ctx, failedCluster.GetScans(), prevSnapshot.GetScans(), scanStore)
		if err != nil {
			return nil, err
		}
		failedClusters[failedCluster.GetClusterId()] = &report.FailedCluster{
			ClusterId:       failedCluster.GetClusterId(),
			ClusterName:     failedCluster.GetClusterName(),
			Reasons:         failedCluster.GetReasons(),
			OperatorVersion: failedCluster.GetOperatorVersion(),
			Scans:           scans,
			Profiles:        failedCluster.GetProfiles(),
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
		data, err := populateScansAndProfiles(ctx, data, reportData, cluster.GetClusterId(), scanStore)
		if err != nil {
			return nil, err
		}
		if failedClusters == nil {
			clusterData[cluster.GetClusterId()] = data
			continue
		}
		if failedCluster, ok := failedClusters[cluster.GetClusterId()]; ok {
			failedCluster.ClusterName = cluster.GetClusterName()
			data.FailedInfo = failedCluster
			data, err = populateFailedScansAndProfiles(ctx, data, reportData, cluster.GetClusterId(), scanStore)
			if err != nil {
				return nil, err
			}
		}
		clusterData[cluster.GetClusterId()] = data
	}
	return clusterData, nil
}

func populateScansAndProfiles(ctx context.Context, data *report.ClusterData, reportData *storage.ComplianceOperatorReportData, clusterID string, scanStore scanDS.DataStore) (*report.ClusterData, error) {
	if data == nil {
		return nil, errors.New("cannot populate scans and profiles of a nil ClusterData")
	}
	scanNamesToProfileNames, err := scanStore.GetProfilesScanNamesByScanConfigAndCluster(ctx, reportData.GetScanConfiguration().GetId(), clusterID)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to retrieve profiles associated with the ScanConfiguration %q in the cluster %q", reportData.GetScanConfiguration().GetId(), clusterID)
	}
	data.Profiles = maps.Values(scanNamesToProfileNames)
	data.Scans = maps.Keys(scanNamesToProfileNames)
	return data, nil
}

func populateFailedScansAndProfiles(ctx context.Context, data *report.ClusterData, reportData *storage.ComplianceOperatorReportData, clusterID string, scanStore scanDS.DataStore) (*report.ClusterData, error) {
	if data.FailedInfo == nil {
		return nil, errors.New("cannot populate scans and profiles of a nil FailedInfo")
	}
	var profileRefIDs []string
	for _, scan := range data.FailedInfo.Scans {
		profileRefIDs = append(profileRefIDs, scan.GetProfile().GetProfileRefId())
	}
	scanNameToProfileName, err := scanStore.GetProfileScanNamesByScanConfigClusterAndProfileRef(ctx, reportData.GetScanConfiguration().GetId(), clusterID, profileRefIDs)
	if err != nil {
		return nil, err
	}
	var profileNames []string
	for _, scan := range data.FailedInfo.Scans {
		if profileName, ok := scanNameToProfileName[scan.GetScanName()]; ok {
			profileNames = append(profileNames, profileName)
		}
	}
	data.FailedInfo.Profiles = profileNames
	return data, nil
}

func populateFailedScans(ctx context.Context, failedScanNames []string, snapshotScans []*storage.ComplianceOperatorReportSnapshotV2_Scan, scanStore scanDS.DataStore) ([]*storage.ComplianceOperatorScanV2, error) {
	scanRefIDs := make([]string, 0, len(snapshotScans))
	for _, scan := range snapshotScans {
		scanRefIDs = append(scanRefIDs, scan.GetScanRefId())
	}
	query := search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanName, failedScanNames...).
		AddExactMatches(search.ComplianceOperatorScanResult, scanRefIDs...).ProtoQuery()
	scans, err := scanStore.SearchScans(ctx, query)
	if err != nil {
		return nil, err
	}
	return scans, nil
}
