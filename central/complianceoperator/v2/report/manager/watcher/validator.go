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
func ValidateScanConfigResults(ctx context.Context, results *ScanConfigWatcherResults, integrationDataStore complianceIntegrationDS.DataStore) (map[string]*storage.ComplianceOperatorReportSnapshotV2_FailedCluster, error) {
	failedClusters := make(map[string]*storage.ComplianceOperatorReportSnapshotV2_FailedCluster)
	errList := errorhelpers.NewErrorList("failed clusters")
	clusterSet := set.NewStringSet()
	for _, scanResult := range results.ScanResults {
		clusterSet.Add(scanResult.Scan.GetClusterId())
		clusterInfo := ValidateScanResults(ctx, scanResult, integrationDataStore)
		if clusterInfo != nil {
			errList.AddError(errors.New(fmt.Sprintf("cluster %s failed", clusterInfo.GetClusterId())))
			failedClusters[clusterInfo.GetClusterId()] = clusterInfo
		}
	}
	// If we have less results than the number of clusters in the scan configuration,
	// we need to add those missing clusters as failed clusters.
	if len(results.ScanConfig.GetClusters()) > len(results.ScanResults) {
		for _, cluster := range results.ScanConfig.GetClusters() {
			if !clusterSet.Contains(cluster.GetClusterId()) {
				clusterInfo := ValidateClusterHealth(ctx, cluster.GetClusterId(), integrationDataStore)
				if clusterInfo != nil {
					errList.AddError(errors.New(fmt.Sprintf("cluster %s failed", clusterInfo.GetClusterId())))
					if len(clusterInfo.Reasons) == 0 {
						clusterInfo.Reasons = []string{report.INTERNAL_ERROR}
					}
					failedClusters[clusterInfo.GetClusterId()] = clusterInfo
				}
			}
		}
	}
	if results.Error != nil {
		if errors.Is(results.Error, ErrScanConfigTimeout) {
			return failedClusters, report.ErrScanConfigWatcherTimeout
		}
		return failedClusters, report.ErrScanWatchersFailed
	}
	if len(failedClusters) > 0 {
		return failedClusters, errList.ToError()
	}
	return failedClusters, nil
}

// ValidateScanResults if there are no errors in the scan results, it returns nil; otherwise it returns the failed cluster information
func ValidateScanResults(ctx context.Context, results *ScanWatcherResults, integrationDataStore complianceIntegrationDS.DataStore) *storage.ComplianceOperatorReportSnapshotV2_FailedCluster {
	if results.Error == nil {
		return nil
	}
	ret := ValidateClusterHealth(ctx, results.Scan.GetClusterId(), integrationDataStore)
	if len(ret.Reasons) > 0 {
		return ret
	}
	ret.Reasons = []string{report.INTERNAL_ERROR}
	if errors.Is(results.Error, ErrScanRemoved) {
		ret.Reasons = []string{fmt.Sprintf(report.SCAN_REMOVED, results.Scan.GetScanName())}
		return ret
	}
	if errors.Is(results.Error, ErrScanTimeout) {
		ret.Reasons = []string{fmt.Sprintf(report.SCAN_TIMEOUT, results.Scan.GetScanName())}
	}
	if checkContextIsDone(results.SensorCtx) {
		ret.Reasons = []string{fmt.Sprintf(report.SCAN_TIMEOUT_SENSOR_DISCONNECTED, results.Scan.GetScanName())}
	}
	return ret
}

// ValidateClusterHealth returns the health status of the Compliance Operator Integration
func ValidateClusterHealth(ctx context.Context, clusterID string, integrationDataStore complianceIntegrationDS.DataStore) *storage.ComplianceOperatorReportSnapshotV2_FailedCluster {
	ret := &storage.ComplianceOperatorReportSnapshotV2_FailedCluster{
		ClusterId:       clusterID,
		OperatorVersion: "",
	}
	coStatus, err := IsComplianceOperatorHealthy(ctx, clusterID, integrationDataStore)
	if errors.Is(err, ErrComplianceOperatorIntegrationDataStore) || errors.Is(err, ErrComplianceOperatorIntegrationZeroIntegrations) {
		ret.Reasons = []string{report.INTERNAL_ERROR}
		return ret
	}
	ret.OperatorVersion = coStatus.GetVersion()
	if errors.Is(err, ErrComplianceOperatorNotInstalled) {
		ret.Reasons = []string{report.COMPLIANCE_NOT_INSTALLED}
		return ret
	}
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
