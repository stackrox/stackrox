package storagetov2

import (
	"github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
)

type checkResultKey struct {
	scanName    string
	profileName string
	checkName   string
}

type scanResultKey struct {
	scanName    string
	profileName string
}

// ComplianceV2CheckResult converts a storage check result to a v2 check result
func ComplianceV2CheckResult(incoming *storage.ComplianceOperatorCheckResultV2) *v2.ComplianceCheckResult {
	converted := &v2.ComplianceCheckResult{
		CheckId:      incoming.GetCheckId(),
		CheckName:    incoming.GetCheckName(),
		Description:  incoming.GetDescription(),
		Instructions: incoming.GetInstructions(),
		Clusters: []*v2.ComplianceCheckResult_ClusterCheckStatus{
			clusterStatus(incoming),
		},
	}

	return converted
}

// ComplianceV2CheckResults converts the storage check results to v2 scan results
func ComplianceV2CheckResults(incoming []*storage.ComplianceOperatorCheckResultV2) []*v2.ComplianceScanResult {
	// Since a check result can hold the status for multiple clusters we need to build from
	// bottom up.  resultsByScanCheck holds check result based on the key combination of
	// scanName, profileName, checkName.  Then when that key is encountered again we
	// add the cluster information to the check result.
	resultsByScanCheck := make(map[checkResultKey]*v2.ComplianceCheckResult)

	// Used to maintain sort order from the query since maps are unordered.
	var orderedKeys []checkResultKey
	for _, result := range incoming {
		key := checkResultKey{
			scanName:    result.GetScanConfigName(),
			profileName: "", // TODO(ROX-20334)
			checkName:   result.GetCheckName(),
		}
		workingResult, found := resultsByScanCheck[key]
		// First time seeing this rule in the results.
		if !found {
			orderedKeys = append(orderedKeys, key)
			resultsByScanCheck[key] = ComplianceV2CheckResult(result)
		} else {
			// Append the new cluster status to the v2 check result.
			workingResult.Clusters = append(workingResult.Clusters, clusterStatus(result))
			resultsByScanCheck[key] = workingResult
		}
	}

	// This builds the outer piece of the result.  The key is simply
	// scan name and profile name.  The individual check results from
	// resultsByScanCheck are appended to build the output
	resultsByScan := make(map[scanResultKey][]*v2.ComplianceCheckResult)
	var scanOrder []scanResultKey
	var convertedResults []*v2.ComplianceScanResult
	for _, key := range orderedKeys {
		scanKey := scanResultKey{
			scanName:    key.scanName,
			profileName: key.profileName,
		}
		result, resultFound := resultsByScanCheck[key]
		if !resultFound {
			continue
		}
		workingResult, found := resultsByScan[scanKey]
		// First time seeing this rule in the results.
		if !found {
			scanOrder = append(scanOrder, scanKey)
			resultsByScan[scanKey] = []*v2.ComplianceCheckResult{result}
		} else {
			workingResult = append(workingResult, result)
			resultsByScan[scanKey] = workingResult
		}
	}

	for _, key := range scanOrder {
		convertedResults = append(convertedResults, &v2.ComplianceScanResult{
			ScanName:     key.scanName,
			ProfileName:  key.profileName,
			CheckResults: resultsByScan[key],
		})
	}

	return convertedResults
}

// ComplianceV2ClusterStats converts the counts to the v2 stats
func ComplianceV2ClusterStats(resultCounts []*datastore.ResourceCountByResultByCluster) []*v2.ComplianceClusterScanStats {
	var convertedResults []*v2.ComplianceClusterScanStats

	for _, resultCount := range resultCounts {
		convertedResults = append(convertedResults, &v2.ComplianceClusterScanStats{
			Cluster: &v2.ComplianceScanCluster{
				ClusterId:   resultCount.ClusterID,
				ClusterName: resultCount.ClusterName,
			},
			ScanStats: &v2.ComplianceScanStatsShim{
				ScanName: resultCount.ScanConfigName,
				CheckStats: []*v2.ComplianceScanStatsShim_ComplianceCheckStatusCount{
					{
						Count:  int32(resultCount.FailCount),
						Status: v2.ComplianceCheckStatus_FAIL,
					},
					{
						Count:  int32(resultCount.InfoCount),
						Status: v2.ComplianceCheckStatus_INFO,
					},
					{
						Count:  int32(resultCount.PassCount),
						Status: v2.ComplianceCheckStatus_PASS,
					},
					{
						Count:  int32(resultCount.ErrorCount),
						Status: v2.ComplianceCheckStatus_ERROR,
					},
					{
						Count:  int32(resultCount.ManualCount),
						Status: v2.ComplianceCheckStatus_MANUAL,
					},
					{
						Count:  int32(resultCount.InconsistentCount),
						Status: v2.ComplianceCheckStatus_INCONSISTENT,
					},
					{
						Count:  int32(resultCount.NotApplicableCount),
						Status: v2.ComplianceCheckStatus_NOT_APPLICABLE,
					},
				},
			},
		})
	}
	return convertedResults
}

func clusterStatus(incoming *storage.ComplianceOperatorCheckResultV2) *v2.ComplianceCheckResult_ClusterCheckStatus {
	return &v2.ComplianceCheckResult_ClusterCheckStatus{
		Cluster: &v2.ComplianceScanCluster{
			ClusterId:   incoming.GetClusterId(),
			ClusterName: incoming.GetClusterName(),
		},
		Status:      convertComplianceCheckStatus(incoming.Status),
		CreatedTime: incoming.GetCreatedTime(),
	}
}
