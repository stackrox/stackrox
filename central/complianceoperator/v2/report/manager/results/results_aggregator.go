package results

import (
	"context"
	"fmt"
	"strings"

	benchmarksDS "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/datastore"
	checkResults "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	"github.com/stackrox/rox/central/complianceoperator/v2/checkresults/utils"
	profileDS "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	remediationDS "github.com/stackrox/rox/central/complianceoperator/v2/remediations/datastore"
	"github.com/stackrox/rox/central/complianceoperator/v2/report"
	complianceRuleDS "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore"
	scanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
)

const (
	DATA_NOT_AVAILABLE = "Data Not Available"
	NO_REMEDIATION     = "No Remediation Available"
)

var (
	log = logging.LoggerForModule()
)

type Aggregator struct {
	checkResultsDS   checkResults.DataStore
	scanDS           scanDS.DataStore
	profileDS        profileDS.DataStore
	remediationDS    remediationDS.DataStore
	benchmarkDS      benchmarksDS.DataStore
	complianceRuleDS complianceRuleDS.DataStore

	aggreateResults aggregateResultsFn
}

func NewAggregator(
	checkResultsDS checkResults.DataStore,
	scanDS scanDS.DataStore,
	profileDS profileDS.DataStore,
	remediationDS remediationDS.DataStore,
	benchmarksDS benchmarksDS.DataStore,
	complianceRuleDS complianceRuleDS.DataStore,
) *Aggregator {
	ret := &Aggregator{
		checkResultsDS:   checkResultsDS,
		scanDS:           scanDS,
		profileDS:        profileDS,
		remediationDS:    remediationDS,
		benchmarkDS:      benchmarksDS,
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
	for _, clusterID := range req.ClusterIDs {
		clusterResults, clusterStatus, err := g.getReportDataForCluster(req.Ctx, req.ScanConfigID, clusterID, req.FailedClusters)
		if err != nil {
			log.Errorf("Data not found for cluster %s", clusterID)
			continue
		}
		resultsCSV[clusterID] = clusterResults
		reportResults.TotalPass += clusterStatus.totalPass
		reportResults.TotalFail += clusterStatus.totalFail
		reportResults.TotalMixed += clusterStatus.totalMixed
	}
	reportResults.Clusters = len(req.ClusterIDs)
	reportResults.Profiles = req.Profiles
	reportResults.ResultCSVs = resultsCSV
	return reportResults
}

func (g *Aggregator) getReportDataForCluster(ctx context.Context, scanConfigID, clusterID string, failedClusters map[string]*storage.ComplianceOperatorReportSnapshotV2_FailedCluster) ([]*report.ResultRow, *checkStatus, error) {
	var ret []*report.ResultRow
	statuses := &checkStatus{
		totalPass:  0,
		totalFail:  0,
		totalMixed: 0,
	}
	// If the cluster is in the failedClusters map, we do not retrieve the data
	if _, ok := failedClusters[clusterID]; ok {
		return ret, statuses, nil
	}
	scanConfigQuery := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorScanConfig, scanConfigID).
		AddExactMatches(search.ClusterID, clusterID).
		ProtoQuery()
	err := g.checkResultsDS.WalkByQuery(ctx, scanConfigQuery, g.aggreateResults(ctx, clusterID, &ret, statuses))
	return ret, statuses, err
}

func (g *Aggregator) AggregateResults(ctx context.Context, clusterID string, clusterResults *[]*report.ResultRow, checkStatus *checkStatus) checkResultWalkByQuery {
	return func(checkResult *storage.ComplianceOperatorCheckResultV2) error {
		row := &report.ResultRow{
			ClusterName:  checkResult.GetClusterName(),
			CheckName:    checkResult.GetCheckName(),
			Description:  checkResult.GetDescription(),
			Status:       checkResult.GetStatus().String(),
			Rationale:    checkResult.GetRationale(),
			Instructions: checkResult.GetInstructions(),
		}
		profileInfo, profileName, err := g.getProfileInfo(ctx, checkResult, clusterID)
		if err != nil {
			return err
		}
		row.Profile = profileInfo
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

func (g *Aggregator) getProfileInfo(ctx context.Context, checkResult *storage.ComplianceOperatorCheckResultV2, clusterID string) (string, string, error) {
	q := search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanRef, checkResult.GetScanRefId()).
		ProtoQuery()
	profiles, err := g.profileDS.SearchProfiles(ctx, q)
	if err != nil {
		return "", "", err
	}
	if len(profiles) < 1 {
		log.Errorf("profile not found for cluster %s and check name %s", clusterID, checkResult.GetCheckName())
		return DATA_NOT_AVAILABLE, "", nil
	}
	return fmt.Sprintf("%s %s", profiles[0].GetName(), profiles[0].GetProfileVersion()), profiles[0].GetName(), nil
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
	controls, err := utils.GetControlsForScanResults(ctx, g.complianceRuleDS, []string{rules[0].GetName()}, profileName, g.benchmarkDS)
	if err != nil {
		log.Errorf("Unable to retrieve controls for result %q.Error %s", checkResult.GetCheckName(), err)
		return DATA_NOT_AVAILABLE, err
	}
	if len(controls) == 0 {
		return DATA_NOT_AVAILABLE, nil
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
