package results

import (
	"context"
	"fmt"
	"strings"
	"time"

	checkResults "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	"github.com/stackrox/rox/central/complianceoperator/v2/checkresults/utils"
	profileDS "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	remediationDS "github.com/stackrox/rox/central/complianceoperator/v2/remediations/datastore"
	"github.com/stackrox/rox/central/complianceoperator/v2/report"
	complianceRuleDS "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore"
	scanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/timestamp"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	DATA_NOT_AVAILABLE = "Data Not Available"
	// CONTROL_NOT_APPLICABLE is used when a control reference cannot be resolved for structural reasons
	// (e.g. tailored profiles with no benchmark mapping, or profiles like E8 whose benchmark short name
	// has no matching standard in rule annotations). DATA_NOT_AVAILABLE is used instead when the lookup
	// fails due to errors or data-integrity issues.
	CONTROL_NOT_APPLICABLE = "N/A"
	NO_REMEDIATION         = "No Remediation Available"
	// NODE_ROLE_NOT_APPLICABLE is used when a check result's scan does not target a specific node
	// role (e.g. a platform scan, or a node scan whose ScanSetting uses the Compliance Operator's
	// "@all" role selector, see https://github.com/ComplianceAsCode/compliance-operator/blob/v1.8.2/pkg/apis/compliance/v1alpha1/scansetting_types.go#L27-L31).
	NODE_ROLE_NOT_APPLICABLE = "N/A"
)

var (
	log = logging.LoggerForModule()
)

type Aggregator struct {
	checkResultsDS   checkResults.DataStore
	scanDS           scanDS.DataStore
	profileDS        profileDS.DataStore
	remediationDS    remediationDS.DataStore
	complianceRuleDS complianceRuleDS.DataStore

	aggreateResults aggregateResultsFn
}

func NewAggregator(
	checkResultsDS checkResults.DataStore,
	scanDS scanDS.DataStore,
	profileDS profileDS.DataStore,
	remediationDS remediationDS.DataStore,
	complianceRuleDS complianceRuleDS.DataStore,
) *Aggregator {
	ret := &Aggregator{
		checkResultsDS:   checkResultsDS,
		scanDS:           scanDS,
		profileDS:        profileDS,
		remediationDS:    remediationDS,
		complianceRuleDS: complianceRuleDS,
	}
	ret.aggreateResults = ret.AggregateResults
	return ret
}

type checkResultWalkByQuery func(*storage.ComplianceOperatorCheckResultV2) error
type aggregateResultsFn func(context.Context, string, *[]*report.ResultRow, *checkStatus) checkResultWalkByQuery

// GetReportData returns map of cluster id and slice of ResultRow
func (g *Aggregator) GetReportData(req *report.Request) *report.Results {
	resultsCSV := make(map[string][]*report.ResultRow)
	reportResults := &report.Results{}
	reportResults.ClustersData = make(map[string]*report.ClusterData)
	for _, clusterID := range req.ClusterIDs {
		clusterData, ok := req.ClusterData[clusterID]
		if !ok {
			log.Errorf("empty cluster data for cluster %q", clusterID)
			continue
		}
		clusterResults, clusterStatus, err := g.getReportDataForCluster(req.Ctx, req.ScanConfigID, clusterID, clusterData)
		if err != nil {
			log.Errorf("Data not found for cluster %s", clusterID)
			continue
		}

		resultsCSV[clusterID] = clusterResults
		reportResults.TotalPass += clusterStatus.totalPass
		reportResults.TotalFail += clusterStatus.totalFail
		reportResults.TotalMixed += clusterStatus.totalMixed
		reportResults.ClustersData[clusterID] = clusterData
	}
	reportResults.Clusters = len(req.ClusterIDs)
	reportResults.Profiles = req.Profiles
	reportResults.ResultCSVs = resultsCSV
	return reportResults
}

func (g *Aggregator) getReportDataForCluster(ctx context.Context, scanConfigID, clusterID string, clusterData *report.ClusterData) ([]*report.ResultRow, *checkStatus, error) {
	var ret []*report.ResultRow
	statuses := &checkStatus{
		totalPass:  0,
		totalFail:  0,
		totalMixed: 0,
	}
	successfulScanNames := clusterData.ScanNames
	if clusterData.FailedInfo != nil {
		if len(clusterData.FailedInfo.FailedScans) == 0 {
			// The entire cluster failed to report (e.g. sensor disconnected,
			// scan config watcher timed out without receiving any results from
			// this cluster). Exclude all results to avoid including stale data
			// from a previous scan cycle in the report.
			return ret, statuses, nil
		}
		allScansSet := set.NewStringSet(successfulScanNames...)
		failedScansSet := set.NewStringSet()
		for _, scan := range clusterData.FailedInfo.FailedScans {
			failedScansSet.Add(scan.GetScanName())
		}
		successfulScanNames = allScansSet.Difference(failedScansSet).AsSlice()
	}
	queryBuilder := search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanConfig, scanConfigID).
		AddExactMatches(search.ClusterID, clusterID).
		AddExactMatches(search.ComplianceOperatorScanName, successfulScanNames...)
	if oldestStartTime := oldestScanStartTime(clusterData.ScanStartTimes); oldestStartTime != nil {
		queryBuilder.AddTimeRangeField(search.ComplianceOperatorCheckLastStartedTime,
			oldestStartTime.AsTime(), timestamp.InfiniteFuture.GoTime())
	}
	scanConfigQuery := queryBuilder.ProtoQuery()
	walkFn := g.aggreateResults(ctx, clusterID, &ret, statuses)
	filteredWalkFn := func(checkResult *storage.ComplianceOperatorCheckResultV2) error {
		if startTime := clusterData.ScanStartTimes[checkResult.GetScanName()]; startTime != nil {
			if checkResult.GetLastStartedTime() == nil || checkResult.GetLastStartedTime().AsTime().Before(startTime.AsTime()) {
				return nil
			}
		}
		return walkFn(checkResult)
	}
	err := g.checkResultsDS.WalkByQuery(ctx, scanConfigQuery, filteredWalkFn)
	return ret, statuses, err
}

func oldestScanStartTime(scanStartTimes map[string]*timestamppb.Timestamp) *timestamppb.Timestamp {
	var oldest *timestamppb.Timestamp
	for _, t := range scanStartTimes {
		if t != nil && (oldest == nil || protoutils.After(oldest, t)) {
			oldest = t
		}
	}
	return oldest
}

func (g *Aggregator) AggregateResults(ctx context.Context, clusterID string, clusterResults *[]*report.ResultRow, checkStatus *checkStatus) checkResultWalkByQuery {
	return func(checkResult *storage.ComplianceOperatorCheckResultV2) error {
		row := &report.ResultRow{
			ClusterName:    checkResult.GetClusterName(),
			CheckName:      checkResult.GetCheckName(),
			Description:    checkResult.GetDescription(),
			Status:         checkResult.GetStatus().String(),
			Rationale:      checkResult.GetRationale(),
			Instructions:   checkResult.GetInstructions(),
			AssessmentTime: protocompat.ConvertTimestampToString(checkResult.GetLastStartedTime(), time.RFC1123),
		}
		profileInfo, profileName, profileType, err := g.getProfileInfo(ctx, checkResult, clusterID)
		if err != nil {
			return err
		}
		row.Profile = profileInfo
		row.ProfileType = profileType
		row.NodeRole = getNodeRole(checkResult, profileName)
		remediationInfo, err := g.getRemediationInfo(ctx, checkResult, clusterID)
		if err != nil {
			return err
		}
		controlsInfo, err := g.getControlsInfo(ctx, checkResult, profileName)
		if err != nil {
			return err
		}
		row.ControlRef = controlsInfo
		row.Remediation = remediationInfo
		*clusterResults = append(*clusterResults, row)
		checkStatus.aggregateCheckResultStatus(checkResult.GetStatus())
		return nil
	}
}

func (g *Aggregator) getProfileInfo(ctx context.Context, checkResult *storage.ComplianceOperatorCheckResultV2, clusterID string) (string, string, string, error) {
	q := search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanRef, checkResult.GetScanRefId()).
		ProtoQuery()
	profiles, err := g.profileDS.SearchProfiles(ctx, q)
	if err != nil {
		return "", "", "", err
	}
	if len(profiles) < 1 {
		log.Errorf("profile not found for cluster %s and check name %s", clusterID, checkResult.GetCheckName())
		return DATA_NOT_AVAILABLE, "", DATA_NOT_AVAILABLE, nil
	}
	profileInfo := fmt.Sprintf("%s %s", profiles[0].GetName(), profiles[0].GetProfileVersion())
	profileType := operatorKindToHumanReadable(profiles[0].GetOperatorKind())
	return profileInfo, profiles[0].GetName(), profileType, nil
}

func operatorKindToHumanReadable(kind storage.ComplianceOperatorProfileV2_OperatorKind) string {
	switch kind {
	case storage.ComplianceOperatorProfileV2_PROFILE:
		return "Profile"
	case storage.ComplianceOperatorProfileV2_TAILORED_PROFILE:
		return "Tailored Profile"
	default:
		return DATA_NOT_AVAILABLE
	}
}

// getNodeRole derives the node role a check's scan targeted from the scan's name. The Compliance
// Operator names a profile's per-role scan "<profileName>-<role>" (one scan per ScanSetting role)
// and leaves the scan name as just "<profileName>" when no specific role is targeted (platform
// scans, or node scans whose ScanSetting uses the Compliance Operator's "@all" role selector).
func getNodeRole(checkResult *storage.ComplianceOperatorCheckResultV2, profileName string) string {
	if profileName == "" {
		return DATA_NOT_AVAILABLE
	}
	role, ok := strings.CutPrefix(checkResult.GetScanName(), profileName+"-")
	if !ok || role == "" {
		return NODE_ROLE_NOT_APPLICABLE
	}
	return role
}

func (g *Aggregator) getRemediationInfo(ctx context.Context, checkResult *storage.ComplianceOperatorCheckResultV2, clusterID string) (string, error) {
	q := search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorCheckName, checkResult.GetCheckName()).
		AddExactMatches(search.ClusterID, checkResult.GetClusterId()).
		ProtoQuery()
	remediations, err := g.remediationDS.SearchRemediations(ctx, q)
	if err != nil {
		log.Errorf("remediations not found for cluster %s and check name %s. Error returned %s", clusterID, checkResult.GetCheckName(), err)
		return DATA_NOT_AVAILABLE, err
	}
	if len(remediations) == 0 {
		return NO_REMEDIATION, nil
	}
	remediationList := make([]string, 0, len(remediations))
	for _, remediation := range remediations {
		remediationList = append(remediationList, remediation.GetName())
	}
	return strings.Join(remediationList, ","), nil
}

func (g *Aggregator) getControlsInfo(ctx context.Context, checkResult *storage.ComplianceOperatorCheckResultV2, profileName string) (string, error) {
	rules, err := g.complianceRuleDS.SearchRules(ctx, search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorRuleRef, checkResult.GetRuleRefId()).ProtoQuery())
	if err != nil {
		log.Errorf("Unable to retrieve compliance rule for result %q", checkResult.GetCheckName())
		return DATA_NOT_AVAILABLE, err
	}
	if len(rules) != 1 {
		// A check result of a cluster maps to a single rule of that same cluster so there should only be 1.
		log.Errorf("Unable to process compliance rule for result %q", checkResult.GetCheckName())
		return DATA_NOT_AVAILABLE, nil
	}
	controls, err := utils.GetControlsForScanResults(ctx, g.complianceRuleDS, []string{rules[0].GetName()}, profileName)
	if err != nil {
		log.Errorf("Unable to retrieve controls for result %q.Error %s", checkResult.GetCheckName(), err)
		return DATA_NOT_AVAILABLE, err
	}
	if len(controls) == 0 {
		return CONTROL_NOT_APPLICABLE, nil
	}
	controlsList := make([]string, 0, len(controls))
	for _, ctrl := range controls {
		controlsList = append(controlsList, fmt.Sprintf("%s %s", ctrl.Standard, ctrl.Control))
	}
	return strings.Join(controlsList, ","), nil
}

type checkStatus struct {
	totalPass  int
	totalMixed int
	totalFail  int
}

func (s *checkStatus) aggregateCheckResultStatus(status storage.ComplianceOperatorCheckResultV2_CheckStatus) {
	switch status {
	case storage.ComplianceOperatorCheckResultV2_PASS:
		s.totalPass += 1
	case storage.ComplianceOperatorCheckResultV2_FAIL:
		s.totalFail += 1
	default:
		s.totalMixed += 1
	}
}
