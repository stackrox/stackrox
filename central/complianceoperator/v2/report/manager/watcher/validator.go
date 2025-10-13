package watcher

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	complianceIntegrationDS "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	"github.com/stackrox/rox/central/complianceoperator/v2/report"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/set"
)

// ValidateScanConfigResults returns a map with the clusters that failed to be scanned.
func ValidateScanConfigResults(ctx context.Context, results *ScanConfigWatcherResults, integrationDataStore complianceIntegrationDS.DataStore) (map[string]*report.FailedCluster, error) {
	failedClusters := make(map[string]*report.FailedCluster)
	errList := errorhelpers.NewErrorList("failed clusters")
	clustersWithResults := set.NewStringSet()
	for _, scanResult := range results.ScanResults {
		clustersWithResults.Add(scanResult.Scan.GetClusterId())
		failedClusterInfo, isInstallationError := ValidateScanResults(ctx, scanResult, integrationDataStore)
		if failedClusterInfo == nil {
			continue
		}
		errList.AddError(errors.New(fmt.Sprintf("scan %s failed in cluster %s", scanResult.Scan.GetScanName(), failedClusterInfo.ClusterId)))
		if previousFailedInfo, ok := failedClusters[failedClusterInfo.ClusterId]; ok && !isInstallationError {
			previousFailedInfo.Reasons = append(previousFailedInfo.Reasons, failedClusterInfo.Reasons...)
			previousFailedInfo.FailedScans = append(previousFailedInfo.FailedScans, failedClusterInfo.FailedScans...)
			continue
		}
		failedClusters[failedClusterInfo.ClusterId] = failedClusterInfo

	}
	// If we have less results than the number of clusters*profiles in the scan configuration,
	// we need to add those missing clusters as failed clusters. *len(results.ScanConfig.GetProfiles())
	if len(results.ScanConfig.GetClusters()) > len(clustersWithResults) {
		for _, cluster := range results.ScanConfig.GetClusters() {
			if clustersWithResults.Contains(cluster.GetClusterId()) {
				continue
			}
			clusterInfo := ValidateClusterHealth(ctx, cluster.GetClusterId(), integrationDataStore)
			errList.AddError(errors.New(fmt.Sprintf("cluster %s failed", clusterInfo.ClusterId)))
			if len(clusterInfo.Reasons) == 0 {
				clusterInfo.Reasons = []string{report.INTERNAL_ERROR}
			}
			failedClusters[clusterInfo.ClusterId] = clusterInfo
		}
	}
	if results.Error != nil && errors.Is(results.Error, ErrScanConfigTimeout) {
		return failedClusters, report.ErrScanConfigWatcherTimeout
	}
	if results.Error != nil {
		return failedClusters, report.ErrScanWatchersFailed
	}
	if len(failedClusters) > 0 {
		return failedClusters, errList.ToError()
	}
	return failedClusters, nil
}

// ValidateScanResults if there are no errors in the scan results, it returns nil; otherwise it returns the failed cluster information
func ValidateScanResults(ctx context.Context, results *ScanWatcherResults, integrationDataStore complianceIntegrationDS.DataStore) (failedCluster *report.FailedCluster, isInstallationError bool) {
	if results.Error == nil {
		return nil, false
	}
	ret := ValidateClusterHealth(ctx, results.Scan.GetClusterId(), integrationDataStore)
	if len(ret.Reasons) > 0 {
		return ret, true
	}
	ret.FailedScans = []*storage.ComplianceOperatorScanV2{results.Scan}
	if errors.Is(results.Error, ErrScanRemoved) {
		ret.Reasons = []string{fmt.Sprintf(report.SCAN_REMOVED_FMT, results.Scan.GetScanName())}
		return ret, false
	}
	if checkContextIsDone(results.SensorCtx) {
		ret.Reasons = []string{fmt.Sprintf(report.SCAN_TIMEOUT_SENSOR_DISCONNECTED_FMT, results.Scan.GetScanName())}
		return ret, false
	}
	if errors.Is(results.Error, ErrScanTimeout) {
		ret.Reasons = []string{fmt.Sprintf(report.SCAN_TIMEOUT_FMT, results.Scan.GetScanName())}
		return ret, false
	}
	ret.Reasons = []string{report.INTERNAL_ERROR}
	return ret, false
}

// ValidateClusterHealth returns the health status of the Compliance Operator Integration
func ValidateClusterHealth(ctx context.Context, clusterID string, integrationDataStore complianceIntegrationDS.DataStore) *report.FailedCluster {
	ret := &report.FailedCluster{
		ClusterId: clusterID,
	}
	coStatus, err := IsComplianceOperatorHealthy(ctx, clusterID, integrationDataStore)
	if errors.Is(err, ErrComplianceOperatorIntegrationDataStore) || errors.Is(err, ErrComplianceOperatorIntegrationZeroIntegrations) {
		ret.Reasons = []string{report.INTERNAL_ERROR}
		return ret
	}
	if errors.Is(err, ErrComplianceOperatorNotInstalled) {
		ret.Reasons = []string{report.COMPLIANCE_NOT_INSTALLED}
		return ret
	}
	ret.OperatorVersion = coStatus.GetVersion()
	if errors.Is(err, ErrComplianceOperatorVersion) {
		ret.Reasons = []string{report.COMPLIANCE_VERSION_ERROR}
		return ret
	}
	return ret
}

func checkContextIsDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
	}
	return false
}
