package storagetov2

import (
	"github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	types "github.com/stackrox/rox/pkg/protocompat"
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
func ComplianceV2CheckResult(incoming *storage.ComplianceOperatorCheckResultV2, lastScanTime *types.Timestamp) *v2.ComplianceClusterCheckStatus {
	converted := &v2.ComplianceClusterCheckStatus{
		CheckId:   incoming.GetCheckId(),
		CheckName: incoming.GetCheckName(),
		Clusters: []*v2.ClusterCheckStatus{
			clusterStatus(incoming, lastScanTime),
		},
		Description:  incoming.GetDescription(),
		Instructions: incoming.GetInstructions(),
		Rationale:    incoming.GetRationale(),
		ValuesUsed:   incoming.GetValuesUsed(),
		Warnings:     incoming.GetWarnings(),
	}

	return converted
}

// ComplianceV2ScanResults converts the storage check results to v2 scan results
func ComplianceV2ScanResults(incoming []*storage.ComplianceOperatorCheckResultV2, scanToScanID map[string]string) []*v2.ComplianceScanResult {
	// Since a check result can hold the status for multiple clusters we need to build from
	// bottom up.  resultsByScanCheck holds check result based on the key combination of
	// scanName, profileName, checkName.  Then when that key is encountered again we
	// add the cluster information to the check result.
	resultsByScanCheck := make(map[checkResultKey]*v2.ComplianceClusterCheckStatus)

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
			resultsByScanCheck[key] = ComplianceV2CheckResult(result, nil)
		} else {
			// Append the new cluster status to the v2 check result.
			workingResult.Clusters = append(workingResult.Clusters, clusterStatus(result, nil))
			resultsByScanCheck[key] = workingResult
		}
	}

	// This builds the outer piece of the result.  The key is simply
	// scan name and profile name.  The individual check results from
	// resultsByScanCheck are appended to build the output
	resultsByScan := make(map[scanResultKey][]*v2.ComplianceClusterCheckStatus)
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
			resultsByScan[scanKey] = []*v2.ComplianceClusterCheckStatus{result}
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

// ComplianceV2ProfileResults converts the counts to the v2 stats
func ComplianceV2ProfileResults(resultCounts []*datastore.ResourceResultsByProfile) []*v2.ComplianceCheckResultStatusCount {
	var profileResults []*v2.ComplianceCheckResultStatusCount

	for _, resultCount := range resultCounts {
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

	return profileResults
}

// ComplianceV2CheckClusterResults converts the storage check results to v2 scan results
func ComplianceV2CheckClusterResults(incoming []*storage.ComplianceOperatorCheckResultV2, lastTimeMap map[string]*types.Timestamp) []*v2.ClusterCheckStatus {
	clusterResults := make([]*v2.ClusterCheckStatus, 0, len(incoming))
	for _, result := range incoming {
		clusterResults = append(clusterResults, clusterStatus(result, lastTimeMap[result.ClusterId]))
	}

	return clusterResults
}

// ComplianceV2CheckResults converts the storage check results to v2 scan results
func ComplianceV2CheckResults(incoming []*storage.ComplianceOperatorCheckResultV2, ruleMap map[string]string) []*v2.ComplianceCheckResult {
	clusterResults := make([]*v2.ComplianceCheckResult, 0, len(incoming))
	for _, result := range incoming {
		clusterResults = append(clusterResults, &v2.ComplianceCheckResult{
			CheckId:      result.GetCheckId(),
			CheckName:    result.GetCheckName(),
			Description:  result.GetDescription(),
			Instructions: result.GetInstructions(),
			Rationale:    result.GetRationale(),
			ValuesUsed:   result.GetValuesUsed(),
			Warnings:     result.GetWarnings(),
			CheckUid:     result.GetId(),
			Status:       convertComplianceCheckStatus(result.Status),
			RuleName:     ruleMap[result.GetRuleRefId()],
		})
	}

	return clusterResults
}

func clusterStatus(incoming *storage.ComplianceOperatorCheckResultV2, lastScanTime *types.Timestamp) *v2.ClusterCheckStatus {
	return &v2.ClusterCheckStatus{
		Cluster: &v2.ComplianceScanCluster{
			ClusterId:   incoming.GetClusterId(),
			ClusterName: incoming.GetClusterName(),
		},
		Status:       convertComplianceCheckStatus(incoming.Status),
		CreatedTime:  incoming.GetCreatedTime(),
		CheckUid:     incoming.GetId(),
		LastScanTime: lastScanTime,
	}
}
