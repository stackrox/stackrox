package storagetov2

import (
	"github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
)

type checkResultKey struct {
	scanConfigName string
	scanConfigID   string
	profileName    string
	checkName      string
}

type scanResultKey struct {
	scanConfigName string
	scanConfigID   string
	profileName    string
}

// ComplianceV2CheckResult converts a storage check result to a v2 check result
func ComplianceV2CheckResult(incoming *storage.ComplianceOperatorCheckResultV2) *v2.ComplianceCheckResult {
	converted := &v2.ComplianceCheckResult{
		CheckId:   incoming.GetCheckId(),
		CheckName: incoming.GetCheckName(),
		Clusters: []*v2.ComplianceCheckResult_ClusterCheckStatus{
			clusterStatus(incoming),
		},
		Description:  incoming.GetDescription(),
		Instructions: incoming.GetInstructions(),
		Rationale:    incoming.GetRationale(),
		ValuesUsed:   incoming.GetValuesUsed(),
		Warnings:     incoming.GetWarnings(),
	}

	return converted
}

// ComplianceV2CheckResults converts the storage check results to v2 scan results
func ComplianceV2CheckResults(incoming []*storage.ComplianceOperatorCheckResultV2, scanToScanID map[string]string) []*v2.ComplianceScanResult {
	// Since a check result can hold the status for multiple clusters we need to build from
	// bottom up.  resultsByScanCheck holds check result based on the key combination of
	// scanName, profileName, checkName.  Then when that key is encountered again we
	// add the cluster information to the check result.
	resultsByScanCheck := make(map[checkResultKey]*v2.ComplianceCheckResult)

	// Used to maintain sort order from the query since maps are unordered.
	var orderedKeys []checkResultKey
	for _, result := range incoming {
		key := checkResultKey{
			scanConfigName: result.GetScanConfigName(),
			scanConfigID:   scanToScanID[result.GetScanConfigName()],
			profileName:    "",
			checkName:      result.GetCheckName(),
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
			scanConfigName: key.scanConfigName,
			scanConfigID:   key.scanConfigID,
			profileName:    key.profileName,
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
			ScanName:     key.scanConfigName,
			ScanConfigId: key.scanConfigID,
			ProfileName:  key.profileName,
			CheckResults: resultsByScan[key],
		})
	}

	return convertedResults
}

// ComplianceV2ClusterStats converts the counts to the v2 stats
func ComplianceV2ClusterStats(resultCounts []*datastore.ResourceResultCountByClusterScan, scanToScanID map[string]string) []*v2.ComplianceClusterScanStats {
	var convertedResults []*v2.ComplianceClusterScanStats

	for _, resultCount := range resultCounts {
		convertedResults = append(convertedResults, &v2.ComplianceClusterScanStats{
			Cluster: &v2.ComplianceScanCluster{
				ClusterId:   resultCount.ClusterID,
				ClusterName: resultCount.ClusterName,
			},
			ScanStats: &v2.ComplianceScanStatsShim{
				ScanName:     resultCount.ScanConfigName,
				ScanConfigId: scanToScanID[resultCount.ScanConfigName],
				CheckStats: []*v2.ComplianceCheckStatusCount{
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

// ComplianceV2ClusterOverallStats converts the counts to the v2 stats
func ComplianceV2ClusterOverallStats(resultCounts []*datastore.ResultStatusCountByCluster, clusterErrors map[string][]string) []*v2.ComplianceClusterOverallStats {
	var convertedResults []*v2.ComplianceClusterOverallStats

	for _, resultCount := range resultCounts {
		convertedResults = append(convertedResults, &v2.ComplianceClusterOverallStats{
			Cluster: &v2.ComplianceScanCluster{
				ClusterId:   resultCount.ClusterID,
				ClusterName: resultCount.ClusterName,
			},
			ClusterErrors: clusterErrors[resultCount.ClusterID],
			CheckStats: []*v2.ComplianceCheckStatusCount{
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
		})
	}
	return convertedResults
}

// ComplianceV2ProfileStats converts the counts to the v2 stats
func ComplianceV2ProfileStats(resultCounts []*datastore.ResourceResultCountByProfile) []*v2.ComplianceProfileScanStats {
	var convertedResults []*v2.ComplianceProfileScanStats

	for _, resultCount := range resultCounts {
		convertedResults = append(convertedResults, &v2.ComplianceProfileScanStats{
			ProfileName: resultCount.ProfileName,
			CheckStats: []*v2.ComplianceCheckStatusCount{
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
		})
	}
	return convertedResults
}

// ComplianceV2ProfileResults converts the counts to the v2 stats
func ComplianceV2ProfileResults(resultCounts []*datastore.ResourceResultsByProfile) *v2.ComplianceProfileResults {
	var profileResults []*v2.ComplianceCheckResultStatusCount

	var profileName string
	for _, resultCount := range resultCounts {
		if profileName == "" {
			profileName = resultCount.ProfileName
		}

		profileResults = append(profileResults, &v2.ComplianceCheckResultStatusCount{
			CheckName: resultCount.CheckName,
			Rationale: resultCount.CheckRationale,
			RuleName:  resultCount.RuleName,
			CheckStats: []*v2.ComplianceCheckStatusCount{
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
		})
	}

	return &v2.ComplianceProfileResults{
		ProfileResults: profileResults,
		ProfileName:    profileName,
	}
}

func clusterStatus(incoming *storage.ComplianceOperatorCheckResultV2) *v2.ComplianceCheckResult_ClusterCheckStatus {
	return &v2.ComplianceCheckResult_ClusterCheckStatus{
		Cluster: &v2.ComplianceScanCluster{
			ClusterId:   incoming.GetClusterId(),
			ClusterName: incoming.GetClusterName(),
		},
		Status:      convertComplianceCheckStatus(incoming.Status),
		CreatedTime: incoming.GetCreatedTime(),
		CheckUid:    incoming.GetId(),
	}
}
