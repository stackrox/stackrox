package results

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/pkg/errors"
	benchmarkMocks "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/datastore/mocks"
	checkResultsMocks "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore/mocks"
	profileMocks "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore/mocks"
	remediationMocks "github.com/stackrox/rox/central/complianceoperator/v2/remediations/datastore/mocks"
	"github.com/stackrox/rox/central/complianceoperator/v2/report"
	"github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore"
	ruleMocks "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore/mocks"
	scanMocks "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	scanConfigID = "scan-config-id"
)

func TestComplianceReportingDataGenerator(t *testing.T) {
	suite.Run(t, new(ComplianceResultsAggregatorSuite))
}

type ComplianceResultsAggregatorSuite struct {
	suite.Suite
	ctrl           *gomock.Controller
	checkResultsDS *checkResultsMocks.MockDataStore
	scanDS         *scanMocks.MockDataStore
	profileDS      *profileMocks.MockDataStore
	remediationDS  *remediationMocks.MockDataStore
	benchmarkDS    *benchmarkMocks.MockDataStore
	ruleDS         *ruleMocks.MockDataStore

	aggregator *Aggregator
}

type getReportDataTestCase struct {
	numClusters               int
	numProfiles               int
	numPassedChecksPerCluster int
	numFailedChecksPerCluster int
	numMixedChecksPerCluster  int
	numFailedClusters         int
	expectedErr               error
}

func (s *ComplianceResultsAggregatorSuite) Test_GetReportData() {
	cases := map[string]getReportDataTestCase{
		"generate report data no error": {
			numClusters:               2,
			numProfiles:               2,
			numPassedChecksPerCluster: 2,
			numFailedChecksPerCluster: 1,
			numMixedChecksPerCluster:  3,
		},
		"generate report data with failed cluster": {
			numClusters:               2,
			numProfiles:               2,
			numPassedChecksPerCluster: 2,
			numFailedChecksPerCluster: 1,
			numMixedChecksPerCluster:  3,
			numFailedClusters:         1,
		},
		"generate report walk by error": {
			numClusters: 3,
			numProfiles: 4,
			expectedErr: errors.New("error"),
		},
	}
	for tname, tcase := range cases {
		s.Run(tname, func() {
			ctx := context.Background()
			req := getRequest(ctx, tcase.numClusters, tcase.numProfiles, tcase.numFailedClusters)
			s.checkResultsDS.EXPECT().WalkByQuery(gomock.Eq(ctx), gomock.Any(), gomock.Any()).
				Times(tcase.numClusters).
				DoAndReturn(func(_, _ any, fn checkResultWalkByQuery) error {
					for i := 0; i < tcase.numPassedChecksPerCluster; i++ {
						_ = fn(&storage.ComplianceOperatorCheckResultV2{
							CheckName: fmt.Sprintf("pass-check-%d", i),
							Status:    storage.ComplianceOperatorCheckResultV2_PASS,
						})
					}
					for i := 0; i < tcase.numFailedChecksPerCluster; i++ {
						_ = fn(&storage.ComplianceOperatorCheckResultV2{
							CheckName: fmt.Sprintf("fail-check-%d", i),
							Status:    storage.ComplianceOperatorCheckResultV2_FAIL,
						})
					}
					for i := 0; i < tcase.numMixedChecksPerCluster; i++ {
						_ = fn(&storage.ComplianceOperatorCheckResultV2{
							CheckName: fmt.Sprintf("mixed-check-%d", i),
							Status:    storage.ComplianceOperatorCheckResultV2_INCONSISTENT,
						})
					}
					return tcase.expectedErr
				})
			s.aggregator.aggreateResults = mockWalkByQueryWrapper
			res := s.aggregator.GetReportData(req)
			assertResults(s.T(), tcase, res)
		})
	}
}

var (
	profiles = []*storage.ComplianceOperatorProfileV2{
		{
			Name:           "profile-1",
			ProfileVersion: "version-profile-1",
		},
	}
	remediations = []*storage.ComplianceOperatorRemediationV2{
		{
			Name: "remediation-1",
		},
	}
	rules = []*storage.ComplianceOperatorRuleV2{
		{
			Name: "rule-1",
		},
	}
	benchmarks = []*storage.ComplianceOperatorBenchmarkV2{
		{
			ShortName: "bench-1",
		},
	}
	controls = []*datastore.ControlResult{
		{
			Standard: "standard-1",
			Control:  "control-1",
		},
	}
)

type walkByQueryTestCase struct {
	check                *storage.ComplianceOperatorCheckResultV2
	expectedProfiles     func() ([]*storage.ComplianceOperatorProfileV2, error)
	expectedRemediations func() ([]*storage.ComplianceOperatorRemediationV2, error)
	expectedRules        func() ([]*storage.ComplianceOperatorRuleV2, error)
	expectedBenchmarks   func() ([]*storage.ComplianceOperatorBenchmarkV2, error)
	expectedControls     func() ([]*datastore.ControlResult, error)
	expectError          bool
}

func (s *ComplianceResultsAggregatorSuite) Test_WalkByQuery() {
	clusterID := "cluster-1"
	cases := map[string]walkByQueryTestCase{
		"pass check no error": {
			check: getCheckResult(storage.ComplianceOperatorCheckResultV2_PASS),
			expectedProfiles: func() ([]*storage.ComplianceOperatorProfileV2, error) {
				return profiles, nil
			},
			expectedRemediations: func() ([]*storage.ComplianceOperatorRemediationV2, error) {
				return remediations, nil
			},
			expectedRules: func() ([]*storage.ComplianceOperatorRuleV2, error) {
				return rules, nil
			},
			expectedBenchmarks: func() ([]*storage.ComplianceOperatorBenchmarkV2, error) {
				return benchmarks, nil
			},
			expectedControls: func() ([]*datastore.ControlResult, error) {
				return controls, nil
			},
		},
		"fail check no error": {
			check: getCheckResult(storage.ComplianceOperatorCheckResultV2_FAIL),
			expectedProfiles: func() ([]*storage.ComplianceOperatorProfileV2, error) {
				return profiles, nil
			},
			expectedRemediations: func() ([]*storage.ComplianceOperatorRemediationV2, error) {
				return remediations, nil
			},
			expectedRules: func() ([]*storage.ComplianceOperatorRuleV2, error) {
				return rules, nil
			},
			expectedBenchmarks: func() ([]*storage.ComplianceOperatorBenchmarkV2, error) {
				return benchmarks, nil
			},
			expectedControls: func() ([]*datastore.ControlResult, error) {
				return controls, nil
			},
		},
		"mixed check no error": {
			check: getCheckResult(storage.ComplianceOperatorCheckResultV2_INCONSISTENT),
			expectedProfiles: func() ([]*storage.ComplianceOperatorProfileV2, error) {
				return profiles, nil
			},
			expectedRemediations: func() ([]*storage.ComplianceOperatorRemediationV2, error) {
				return remediations, nil
			},
			expectedRules: func() ([]*storage.ComplianceOperatorRuleV2, error) {
				return rules, nil
			},
			expectedBenchmarks: func() ([]*storage.ComplianceOperatorBenchmarkV2, error) {
				return benchmarks, nil
			},
			expectedControls: func() ([]*datastore.ControlResult, error) {
				return controls, nil
			},
		},
		"profile search error": {
			check: getCheckResult(storage.ComplianceOperatorCheckResultV2_PASS),
			expectedProfiles: func() ([]*storage.ComplianceOperatorProfileV2, error) {
				return nil, errors.New("error")
			},
			expectError: true,
		},
		"profile not found": {
			check: getCheckResult(storage.ComplianceOperatorCheckResultV2_PASS),
			expectedProfiles: func() ([]*storage.ComplianceOperatorProfileV2, error) {
				return []*storage.ComplianceOperatorProfileV2{}, nil
			},
			expectedRemediations: func() ([]*storage.ComplianceOperatorRemediationV2, error) {
				return remediations, nil
			},
			expectedRules: func() ([]*storage.ComplianceOperatorRuleV2, error) {
				return rules, nil
			},
			expectedBenchmarks: func() ([]*storage.ComplianceOperatorBenchmarkV2, error) {
				return benchmarks, nil
			},
			expectedControls: func() ([]*datastore.ControlResult, error) {
				return controls, nil
			},
		},
		"remediation search error": {
			check: getCheckResult(storage.ComplianceOperatorCheckResultV2_PASS),
			expectedProfiles: func() ([]*storage.ComplianceOperatorProfileV2, error) {
				return profiles, nil
			},
			expectedRemediations: func() ([]*storage.ComplianceOperatorRemediationV2, error) {
				return nil, errors.New("error")
			},
			expectError: true,
		},
		"remediation not found": {
			check: getCheckResult(storage.ComplianceOperatorCheckResultV2_PASS),
			expectedProfiles: func() ([]*storage.ComplianceOperatorProfileV2, error) {
				return profiles, nil
			},
			expectedRemediations: func() ([]*storage.ComplianceOperatorRemediationV2, error) {
				return []*storage.ComplianceOperatorRemediationV2{}, nil
			},
			expectedRules: func() ([]*storage.ComplianceOperatorRuleV2, error) {
				return rules, nil
			},
			expectedBenchmarks: func() ([]*storage.ComplianceOperatorBenchmarkV2, error) {
				return benchmarks, nil
			},
			expectedControls: func() ([]*datastore.ControlResult, error) {
				return controls, nil
			},
		},
		"rule search error": {
			check: getCheckResult(storage.ComplianceOperatorCheckResultV2_PASS),
			expectedProfiles: func() ([]*storage.ComplianceOperatorProfileV2, error) {
				return profiles, nil
			},
			expectedRemediations: func() ([]*storage.ComplianceOperatorRemediationV2, error) {
				return remediations, nil
			},
			expectedRules: func() ([]*storage.ComplianceOperatorRuleV2, error) {
				return nil, errors.New("error")
			},
			expectError: true,
		},
		"rule not found": {
			check: getCheckResult(storage.ComplianceOperatorCheckResultV2_PASS),
			expectedProfiles: func() ([]*storage.ComplianceOperatorProfileV2, error) {
				return profiles, nil
			},
			expectedRemediations: func() ([]*storage.ComplianceOperatorRemediationV2, error) {
				return remediations, nil
			},
			expectedRules: func() ([]*storage.ComplianceOperatorRuleV2, error) {
				return []*storage.ComplianceOperatorRuleV2{}, nil
			},
		},
		"benchmark search error": {
			check: getCheckResult(storage.ComplianceOperatorCheckResultV2_PASS),
			expectedProfiles: func() ([]*storage.ComplianceOperatorProfileV2, error) {
				return profiles, nil
			},
			expectedRemediations: func() ([]*storage.ComplianceOperatorRemediationV2, error) {
				return remediations, nil
			},
			expectedRules: func() ([]*storage.ComplianceOperatorRuleV2, error) {
				return rules, nil
			},
			expectedBenchmarks: func() ([]*storage.ComplianceOperatorBenchmarkV2, error) {
				return nil, errors.New("error")
			},
			expectError: true,
		},
		"benchmark not found": {
			check: getCheckResult(storage.ComplianceOperatorCheckResultV2_PASS),
			expectedProfiles: func() ([]*storage.ComplianceOperatorProfileV2, error) {
				return profiles, nil
			},
			expectedRemediations: func() ([]*storage.ComplianceOperatorRemediationV2, error) {
				return remediations, nil
			},
			expectedRules: func() ([]*storage.ComplianceOperatorRuleV2, error) {
				return rules, nil
			},
			expectedBenchmarks: func() ([]*storage.ComplianceOperatorBenchmarkV2, error) {
				return []*storage.ComplianceOperatorBenchmarkV2{}, nil
			},
		},
		"control search error": {
			check: getCheckResult(storage.ComplianceOperatorCheckResultV2_PASS),
			expectedProfiles: func() ([]*storage.ComplianceOperatorProfileV2, error) {
				return profiles, nil
			},
			expectedRemediations: func() ([]*storage.ComplianceOperatorRemediationV2, error) {
				return remediations, nil
			},
			expectedRules: func() ([]*storage.ComplianceOperatorRuleV2, error) {
				return rules, nil
			},
			expectedBenchmarks: func() ([]*storage.ComplianceOperatorBenchmarkV2, error) {
				return benchmarks, nil
			},
			expectedControls: func() ([]*datastore.ControlResult, error) {
				return nil, errors.New("error")
			},
			expectError: true,
		},
		"control not found": {
			check: getCheckResult(storage.ComplianceOperatorCheckResultV2_PASS),
			expectedProfiles: func() ([]*storage.ComplianceOperatorProfileV2, error) {
				return profiles, nil
			},
			expectedRemediations: func() ([]*storage.ComplianceOperatorRemediationV2, error) {
				return remediations, nil
			},
			expectedRules: func() ([]*storage.ComplianceOperatorRuleV2, error) {
				return rules, nil
			},
			expectedBenchmarks: func() ([]*storage.ComplianceOperatorBenchmarkV2, error) {
				return benchmarks, nil
			},
			expectedControls: func() ([]*datastore.ControlResult, error) {
				return []*datastore.ControlResult{}, nil
			},
		},
	}
	for tname, tcase := range cases {
		s.Run(tname, func() {
			if tcase.expectedProfiles != nil {
				s.profileDS.EXPECT().SearchProfiles(gomock.Any(), gomock.Any()).Times(1).Return(tcase.expectedProfiles())
			}
			if tcase.expectedRemediations != nil {
				s.remediationDS.EXPECT().SearchRemediations(gomock.Any(), gomock.Any()).Times(1).Return(tcase.expectedRemediations())
			}
			if tcase.expectedRules != nil {
				s.ruleDS.EXPECT().SearchRules(gomock.Any(), gomock.Any()).Times(1).Return(tcase.expectedRules())
			}
			if tcase.expectedBenchmarks != nil {
				s.benchmarkDS.EXPECT().GetBenchmarksByProfileName(gomock.Any(), gomock.Any()).Times(1).Return(tcase.expectedBenchmarks())
			}
			if tcase.expectedControls != nil {
				s.ruleDS.EXPECT().GetControlsByRulesAndBenchmarks(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(tcase.expectedControls())
			}
			var results []*report.ResultRow
			status := &checkStatus{}
			err := s.aggregator.AggregateResults(context.Background(), clusterID, &results, status)(tcase.check)
			if tcase.expectError {
				assert.Error(s.T(), err)
			} else {
				assert.NoError(s.T(), err)
				assertStatus(s.T(), tcase.check.GetStatus(), status)
				require.Len(s.T(), results, 1)
				assertResult(s.T(), tcase, results[0])
			}
		})
	}
}

func (s *ComplianceResultsAggregatorSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.checkResultsDS = checkResultsMocks.NewMockDataStore(s.ctrl)
	s.scanDS = scanMocks.NewMockDataStore(s.ctrl)
	s.profileDS = profileMocks.NewMockDataStore(s.ctrl)
	s.remediationDS = remediationMocks.NewMockDataStore(s.ctrl)
	s.benchmarkDS = benchmarkMocks.NewMockDataStore(s.ctrl)
	s.ruleDS = ruleMocks.NewMockDataStore(s.ctrl)

	s.aggregator = NewAggregator(s.checkResultsDS, s.scanDS, s.profileDS, s.remediationDS, s.benchmarkDS, s.ruleDS)
}

func getRequest(ctx context.Context, numClusters, numProfiles, numFailedClusters int) *report.Request {
	ret := &report.Request{
		Ctx:          ctx,
		ScanConfigID: scanConfigID,
		ClusterIDs:   getNames("cluster", numClusters),
		Profiles:     getNames("profile", numProfiles),
	}
	if numFailedClusters > 0 {
		failedClusters := make(map[string]*storage.ComplianceOperatorReportSnapshotV2_FailedCluster)
		for i := numClusters; i < numFailedClusters+numClusters; i++ {
			id := fmt.Sprintf("cluster-%d", i)
			ret.ClusterIDs = append(ret.ClusterIDs, id)
			failedClusters[id] = &storage.ComplianceOperatorReportSnapshotV2_FailedCluster{}
		}
		ret.FailedClusters = failedClusters
	}
	return ret
}

func getNames(prefix string, num int) []string {
	ret := make([]string, 0, 2)
	for i := 0; i < num; i++ {
		ret = append(ret, fmt.Sprintf("%s-%d", prefix, i))
	}
	return ret
}

func mockWalkByQueryWrapper(_ context.Context, clusterID string, clusterResults *[]*report.ResultRow, status *checkStatus) checkResultWalkByQuery {
	return func(check *storage.ComplianceOperatorCheckResultV2) error {
		status.aggregateCheckResultStatus(check.GetStatus())
		*clusterResults = append(*clusterResults, getRowFromCluster(check.GetCheckName(), clusterID))
		return nil
	}
}

func getRowFromCluster(check, clusterID string) *report.ResultRow {
	return &report.ResultRow{
		ClusterName:  clusterID,
		CheckName:    fmt.Sprintf("check-%s-%s", clusterID, check),
		Description:  fmt.Sprintf("description-%s-%s", clusterID, check),
		Status:       fmt.Sprintf("status-%s-%s", clusterID, check),
		Rationale:    fmt.Sprintf("rationale-%s-%s", clusterID, check),
		Instructions: fmt.Sprintf("instructions-%s-%s", clusterID, check),
		Profile:      fmt.Sprintf("profile-%s-%s", clusterID, check),
		ControlRef:   fmt.Sprintf("control-%s-%s", clusterID, check),
		Remediation:  fmt.Sprintf("remediation=%s-%s", clusterID, check),
	}
}

func assertResults(t *testing.T, tcase getReportDataTestCase, res *report.Results) {
	assert.Equal(t, tcase.numClusters+tcase.numFailedClusters, res.Clusters)
	assert.Equal(t, tcase.numProfiles, len(res.Profiles))
	if tcase.expectedErr != nil {
		assert.Equal(t, 0, res.TotalPass)
		assert.Equal(t, 0, res.TotalFail)
		assert.Equal(t, 0, res.TotalMixed)
		assert.Len(t, res.ResultCSVs, 0)
		return
	}
	assert.Equal(t, tcase.numPassedChecksPerCluster*tcase.numClusters, res.TotalPass)
	assert.Equal(t, tcase.numFailedChecksPerCluster*tcase.numClusters, res.TotalFail)
	assert.Equal(t, tcase.numMixedChecksPerCluster*tcase.numClusters, res.TotalMixed)
	for i := 0; i < tcase.numClusters; i++ {
		clusterID := fmt.Sprintf("cluster-%d", i)
		var expResults []*report.ResultRow
		for j := 0; j < tcase.numPassedChecksPerCluster; j++ {
			row := getRowFromCluster(fmt.Sprintf("pass-check-%d", j), clusterID)
			expResults = append(expResults, row)
		}
		for j := 0; j < tcase.numFailedChecksPerCluster; j++ {
			row := getRowFromCluster(fmt.Sprintf("fail-check-%d", j), clusterID)
			expResults = append(expResults, row)
		}
		for j := 0; j < tcase.numMixedChecksPerCluster; j++ {
			row := getRowFromCluster(fmt.Sprintf("mixed-check-%d", j), clusterID)
			expResults = append(expResults, row)
		}
		assert.Equal(t, expResults, res.ResultCSVs[clusterID])
	}
}

func getCheckResult(status storage.ComplianceOperatorCheckResultV2_CheckStatus) *storage.ComplianceOperatorCheckResultV2 {
	return &storage.ComplianceOperatorCheckResultV2{
		ClusterName:  "cluster-1",
		CheckName:    "check",
		Description:  "description",
		Status:       status,
		Rationale:    "rationale",
		Instructions: "instructions",
	}
}

func assertStatus(t *testing.T, expected storage.ComplianceOperatorCheckResultV2_CheckStatus, actual *checkStatus) {
	switch expected {
	case storage.ComplianceOperatorCheckResultV2_PASS:
		assert.Equal(t, 1, actual.totalPass)
		assert.Equal(t, 0, actual.totalFail)
		assert.Equal(t, 0, actual.totalMixed)
	case storage.ComplianceOperatorCheckResultV2_FAIL:
		assert.Equal(t, 0, actual.totalPass)
		assert.Equal(t, 1, actual.totalFail)
		assert.Equal(t, 0, actual.totalMixed)
	default:
		assert.Equal(t, 0, actual.totalPass)
		assert.Equal(t, 0, actual.totalFail)
		assert.Equal(t, 1, actual.totalMixed)
	}
}

func assertResult(t *testing.T, tcase walkByQueryTestCase, row *report.ResultRow) {
	assert.Equal(t, tcase.check.GetClusterName(), row.ClusterName)
	assert.Equal(t, tcase.check.GetCheckName(), row.CheckName)
	assert.Equal(t, tcase.check.GetDescription(), row.Description)
	assert.Equal(t, tcase.check.GetStatus().String(), row.Status)
	assert.Equal(t, tcase.check.GetRationale(), row.Rationale)
	assert.Equal(t, tcase.check.GetInstructions(), row.Instructions)
	if tcase.expectedProfiles != nil {
		expProfiles, _ := tcase.expectedProfiles()
		if len(expProfiles) < 1 {
			assert.Equal(t, DATA_NOT_AVAILABLE, row.Profile)
		} else {
			require.Len(t, expProfiles, 1)
			assert.Equal(t, fmt.Sprintf("%s %s", expProfiles[0].GetName(), expProfiles[0].GetProfileVersion()), row.Profile)
		}
	}
	if tcase.expectedRemediations != nil {
		expRemediations, _ := tcase.expectedRemediations()
		if len(expRemediations) == 0 {
			assert.Equal(t, NO_REMEDIATION, row.Remediation)
		} else {
			expRemediationNames := make([]string, 0, len(remediations))
			for _, remediation := range expRemediations {
				expRemediationNames = append(expRemediationNames, remediation.GetName())
			}
			assert.Equal(t, strings.Join(expRemediationNames, ","), row.Remediation)
		}
	}
	if tcase.expectedRules == nil {
		return
	}
	expRules, _ := tcase.expectedRules()
	if len(expRules) != 1 {
		assert.Equal(t, DATA_NOT_AVAILABLE, row.ControlRef)
		return
	}
	if tcase.expectedBenchmarks == nil {
		assert.Equal(t, DATA_NOT_AVAILABLE, row.ControlRef)
		return
	}
	expBench, _ := tcase.expectedBenchmarks()
	if len(expBench) == 0 {
		assert.Equal(t, DATA_NOT_AVAILABLE, row.ControlRef)
		return
	}
	if tcase.expectedControls == nil {
		return
	}
	expControls, _ := tcase.expectedControls()
	if len(expControls) == 0 {
		assert.Equal(t, DATA_NOT_AVAILABLE, row.ControlRef)
		return
	}
	expControlInfos := make([]string, 0, len(expControls))
	for _, c := range expControls {
		expControlInfos = append(expControlInfos, fmt.Sprintf("%s %s", c.Standard, c.Control))
	}
	assert.Equal(t, strings.Join(expControlInfos, ","), row.ControlRef)
}
