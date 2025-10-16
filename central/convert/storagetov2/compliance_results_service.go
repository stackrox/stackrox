package storagetov2

import (
	"github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	compRule "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore"
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
func ComplianceV2CheckResult(incoming *storage.ComplianceOperatorCheckResultV2, lastScanTime *types.Timestamp, ruleName string, controlResults []*compRule.ControlResult) *v2.ComplianceClusterCheckStatus {
	converted := &v2.ComplianceClusterCheckStatus{}
	converted.SetCheckId(incoming.GetCheckId())
	converted.SetCheckName(incoming.GetCheckName())
	converted.SetClusters([]*v2.ClusterCheckStatus{
		clusterStatus(incoming, lastScanTime),
	})
	converted.SetDescription(incoming.GetDescription())
	converted.SetInstructions(incoming.GetInstructions())
	converted.SetRationale(incoming.GetRationale())
	converted.SetValuesUsed(incoming.GetValuesUsed())
	converted.SetWarnings(incoming.GetWarnings())
	converted.SetLabels(incoming.GetLabels())
	converted.SetAnnotations(incoming.GetAnnotations())
	converted.SetControls(GetControls(ruleName, controlResults))

	return converted
}

// ComplianceV2SpecificCheckResult converts a storage check result to a v2 check result
func ComplianceV2SpecificCheckResult(incoming []*storage.ComplianceOperatorCheckResultV2, checkName string, controls []*v2.ComplianceControl) *v2.ComplianceClusterCheckStatus {
	var converted *v2.ComplianceClusterCheckStatus
	for _, result := range incoming {
		if result.GetCheckName() != checkName {
			continue
		}

		if converted == nil {
			converted = &v2.ComplianceClusterCheckStatus{}
			converted.SetCheckId(result.GetCheckId())
			converted.SetCheckName(result.GetCheckName())
			converted.SetClusters([]*v2.ClusterCheckStatus{
				clusterStatus(result, nil),
			})
			converted.SetDescription(result.GetDescription())
			converted.SetInstructions(result.GetInstructions())
			converted.SetRationale(result.GetRationale())
			converted.SetValuesUsed(result.GetValuesUsed())
			converted.SetWarnings(result.GetWarnings())
			converted.SetLabels(result.GetLabels())
			converted.SetAnnotations(result.GetAnnotations())
			converted.SetControls(controls)
		} else {
			converted.SetClusters(append(converted.GetClusters(), clusterStatus(result, nil)))
		}
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
			resultsByScanCheck[key] = ComplianceV2CheckResult(result, nil, "", nil)
		} else {
			// Append the new cluster status to the v2 check result.
			workingResult.SetClusters(append(workingResult.GetClusters(), clusterStatus(result, nil)))
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
		csr := &v2.ComplianceScanResult{}
		csr.SetScanName(key.scanConfigName)
		csr.SetScanConfigId(key.scanConfigID)
		csr.SetProfileName(key.profileName)
		csr.SetCheckResults(resultsByScan[key])
		convertedResults = append(convertedResults, csr)
	}

	return convertedResults
}

// ComplianceV2ProfileResults converts the counts to the v2 stats
func ComplianceV2ProfileResults(resultCounts []*datastore.ResourceResultsByProfile, controlResults []*compRule.ControlResult) []*v2.ComplianceCheckResultStatusCount {
	var profileResults []*v2.ComplianceCheckResultStatusCount

	for _, resultCount := range resultCounts {
		controls := GetControls(resultCount.RuleName, controlResults)

		profileResults = append(profileResults, v2.ComplianceCheckResultStatusCount_builder{
			CheckName: resultCount.CheckName,
			Rationale: resultCount.CheckRationale,
			RuleName:  resultCount.RuleName,
			Controls:  controls,
			CheckStats: []*v2.ComplianceCheckStatusCount{
				v2.ComplianceCheckStatusCount_builder{
					Count:  int32(resultCount.FailCount),
					Status: v2.ComplianceCheckStatus_FAIL,
				}.Build(),
				v2.ComplianceCheckStatusCount_builder{
					Count:  int32(resultCount.InfoCount),
					Status: v2.ComplianceCheckStatus_INFO,
				}.Build(),
				v2.ComplianceCheckStatusCount_builder{
					Count:  int32(resultCount.PassCount),
					Status: v2.ComplianceCheckStatus_PASS,
				}.Build(),
				v2.ComplianceCheckStatusCount_builder{
					Count:  int32(resultCount.ErrorCount),
					Status: v2.ComplianceCheckStatus_ERROR,
				}.Build(),
				v2.ComplianceCheckStatusCount_builder{
					Count:  int32(resultCount.ManualCount),
					Status: v2.ComplianceCheckStatus_MANUAL,
				}.Build(),
				v2.ComplianceCheckStatusCount_builder{
					Count:  int32(resultCount.InconsistentCount),
					Status: v2.ComplianceCheckStatus_INCONSISTENT,
				}.Build(),
				v2.ComplianceCheckStatusCount_builder{
					Count:  int32(resultCount.NotApplicableCount),
					Status: v2.ComplianceCheckStatus_NOT_APPLICABLE,
				}.Build(),
			},
		}.Build())
	}

	return profileResults
}

// ComplianceV2CheckClusterResults converts the storage check results to v2 scan results
func ComplianceV2CheckClusterResults(incoming []*storage.ComplianceOperatorCheckResultV2, lastTimeMap map[string]*types.Timestamp) []*v2.ClusterCheckStatus {
	clusterResults := make([]*v2.ClusterCheckStatus, 0, len(incoming))
	for _, result := range incoming {
		clusterResults = append(clusterResults, clusterStatus(result, lastTimeMap[result.GetClusterId()]))
	}

	return clusterResults
}

// ComplianceV2CheckResults converts the storage check results to v2 scan results
func ComplianceV2CheckResults(incoming []*storage.ComplianceOperatorCheckResultV2, ruleMap map[string]string, controlResults []*compRule.ControlResult) []*v2.ComplianceCheckResult {
	clusterResults := make([]*v2.ComplianceCheckResult, 0, len(incoming))
	for _, result := range incoming {
		clusterResults = append(clusterResults, checkResult(result, ruleMap[result.GetRuleRefId()], controlResults))
	}

	return clusterResults
}

func ComplianceV2CheckData(incoming []*storage.ComplianceOperatorCheckResultV2, ruleMap map[string]string, controlMap map[string][]*compRule.ControlResult) []*v2.ComplianceCheckData {
	results := make([]*v2.ComplianceCheckData, 0, len(incoming))
	for _, result := range incoming {
		ccd := &v2.ComplianceCheckData{}
		ccd.SetClusterId(result.GetClusterId())
		ccd.SetScanName(result.GetScanConfigName())
		ccd.SetResult(checkResult(result, ruleMap[result.GetRuleRefId()], controlMap[result.GetCheckName()]))
		results = append(results, ccd)
	}

	return results
}

func clusterStatus(incoming *storage.ComplianceOperatorCheckResultV2, lastScanTime *types.Timestamp) *v2.ClusterCheckStatus {
	csc := &v2.ComplianceScanCluster{}
	csc.SetClusterId(incoming.GetClusterId())
	csc.SetClusterName(incoming.GetClusterName())
	ccs := &v2.ClusterCheckStatus{}
	ccs.SetCluster(csc)
	ccs.SetStatus(convertComplianceCheckStatus(incoming.GetStatus()))
	ccs.SetCreatedTime(incoming.GetCreatedTime())
	ccs.SetCheckUid(incoming.GetId())
	ccs.SetLastScanTime(lastScanTime)
	return ccs
}

func checkResult(incoming *storage.ComplianceOperatorCheckResultV2, ruleName string, controlResults []*compRule.ControlResult) *v2.ComplianceCheckResult {
	ccr := &v2.ComplianceCheckResult{}
	ccr.SetCheckId(incoming.GetCheckId())
	ccr.SetCheckName(incoming.GetCheckName())
	ccr.SetCheckUid(incoming.GetId())
	ccr.SetDescription(incoming.GetDescription())
	ccr.SetInstructions(incoming.GetInstructions())
	ccr.SetControls(GetControls(ruleName, controlResults))
	ccr.SetRationale(incoming.GetRationale())
	ccr.SetValuesUsed(incoming.GetValuesUsed())
	ccr.SetWarnings(incoming.GetWarnings())
	ccr.SetStatus(convertComplianceCheckStatus(incoming.GetStatus()))
	ccr.SetRuleName(ruleName)
	ccr.SetLabels(incoming.GetLabels())
	ccr.SetAnnotations(incoming.GetAnnotations())
	return ccr
}

func GetControls(ruleName string, controlResults []*compRule.ControlResult) []*v2.ComplianceControl {
	var controls []*v2.ComplianceControl
	for _, controlResult := range controlResults {
		if controlResult.RuleName == ruleName {
			cc := &v2.ComplianceControl{}
			cc.SetStandard(controlResult.Standard)
			cc.SetControl(controlResult.Control)
			controls = append(controls, cc)
		}
	}

	return controls
}
