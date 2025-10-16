package service

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	benchmarkMocks "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/datastore/mocks"
	"github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	resultMocks "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore/mocks"
	integrationMocks "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore/mocks"
	profileDatastore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore/mocks"
	ruleDS "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore"
	ruleMocks "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore/mocks"
	scanConfigMocks "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore/mocks"
	scanMocks "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore/mocks"
	convertUtils "github.com/stackrox/rox/central/convert/testutils"
	v1 "github.com/stackrox/rox/generated/api/v1"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/protoassert"
	types "github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	scan1 = &storage.ComplianceOperatorScanV2{
		Id:               "",
		ClusterId:        testconsts.Cluster1,
		LastExecutedTime: types.TimestampNow(),
	}
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

func TestComplianceResultsService(t *testing.T) {
	suite.Run(t, new(ComplianceResultsServiceTestSuite))
}

type ComplianceResultsServiceTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	ctx             context.Context
	resultDatastore *resultMocks.MockDataStore
	scanConfigDS    *scanConfigMocks.MockDataStore
	integrationDS   *integrationMocks.MockDataStore
	ruleDS          *ruleMocks.MockDataStore
	service         Service
	profilsDS       *profileDatastore.MockDataStore
	scanDS          *scanMocks.MockDataStore
	benchmarkDS     *benchmarkMocks.MockDataStore
}

func (s *ComplianceResultsServiceTestSuite) SetupSuite() {
	s.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")
	if !features.ComplianceEnhancements.Enabled() {
		s.T().Skip("Skip test when compliance enhancements are disabled")
		s.T().SkipNow()
	}

	s.ctx = sac.WithAllAccess(context.Background())
}

func (s *ComplianceResultsServiceTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.resultDatastore = resultMocks.NewMockDataStore(s.mockCtrl)
	s.scanConfigDS = scanConfigMocks.NewMockDataStore(s.mockCtrl)
	s.integrationDS = integrationMocks.NewMockDataStore(s.mockCtrl)
	s.profilsDS = profileDatastore.NewMockDataStore(s.mockCtrl)
	s.ruleDS = ruleMocks.NewMockDataStore(s.mockCtrl)
	s.scanDS = scanMocks.NewMockDataStore(s.mockCtrl)
	s.benchmarkDS = benchmarkMocks.NewMockDataStore(s.mockCtrl)

	s.service = New(s.resultDatastore, s.scanConfigDS, s.integrationDS, s.profilsDS, s.ruleDS, s.scanDS, s.benchmarkDS)
}

func (s *ComplianceResultsServiceTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ComplianceResultsServiceTestSuite) TestGetComplianceScanResults() {
	testCases := []struct {
		desc         string
		query        *apiV2.RawQuery
		expectedResp []*apiV2.ComplianceCheckData
		expectedErr  error
		found        bool
		setMocks     func()
	}{
		{
			desc:         "Empty query",
			query:        apiV2.RawQuery_builder{Query: ""}.Build(),
			expectedErr:  nil,
			expectedResp: convertUtils.GetConvertedComplianceData(s.T()),
			found:        true,
			setMocks: func() {
				expectedQ := search.EmptyQuery()
				countQuery := expectedQ.CloneVT()
				expectedQ.SetPagination(v1.QueryPagination_builder{Limit: maxPaginationLimit}.Build())

				storageResults := convertUtils.GetComplianceStorageResults(s.T())
				s.resultDatastore.EXPECT().SearchComplianceCheckResults(gomock.Any(), expectedQ).Return(storageResults, nil).Times(1)
				s.ruleDS.EXPECT().SearchRules(gomock.Any(), gomock.Any()).Return([]*storage.ComplianceOperatorRuleV2{storage.ComplianceOperatorRuleV2_builder{Name: "rule-name"}.Build()}, nil).AnyTimes()
				s.resultDatastore.EXPECT().CountCheckResults(gomock.Any(), countQuery).Return(7, nil).Times(1)

				profilesOcp := []*storage.ComplianceOperatorProfileV2{storage.ComplianceOperatorProfileV2_builder{
					Name:           "ocp4",
					ProfileVersion: "test_version_ocp4",
					Title:          "test_title_ocp4",
				}.Build()}
				s.profilsDS.EXPECT().SearchProfiles(gomock.Any(), search.NewQueryBuilder().
					AddExactMatches(search.ComplianceOperatorScanRef, "test-ref").ProtoQuery()).Return(profilesOcp, nil).AnyTimes()
				s.benchmarkDS.EXPECT().GetBenchmarksByProfileName(gomock.Any(), "ocp4").Return(fixtures.GetExpectedBenchmark(), nil).AnyTimes()
				s.ruleDS.EXPECT().GetControlsByRulesAndBenchmarks(gomock.Any(), []string{"rule-name"}, []string{"OCP_CIS"}).Return(getExpectedControlResults(), nil).AnyTimes()
			},
		},
		{
			desc:        "Query with search field",
			query:       apiV2.RawQuery_builder{Query: "Cluster ID:" + fixtureconsts.Cluster1}.Build(),
			expectedErr: nil,
			found:       true,
			setMocks: func() {
				expectedQ := search.NewQueryBuilder().AddStrings(search.ClusterID, fixtureconsts.Cluster1).ProtoQuery()
				countQuery := expectedQ.CloneVT()
				expectedQ.SetPagination(v1.QueryPagination_builder{Limit: maxPaginationLimit}.Build())

				storageResults := convertUtils.GetOneClusterComplianceStorageResults(s.T(), fixtureconsts.Cluster1)
				s.resultDatastore.EXPECT().SearchComplianceCheckResults(gomock.Any(), expectedQ).Return(storageResults, nil).Times(1)
				s.ruleDS.EXPECT().SearchRules(gomock.Any(), gomock.Any()).Return([]*storage.ComplianceOperatorRuleV2{storage.ComplianceOperatorRuleV2_builder{Name: "rule-name"}.Build()}, nil).AnyTimes()
				s.resultDatastore.EXPECT().CountCheckResults(gomock.Any(), countQuery).Return(7, nil).Times(1)

				profilesOcp := []*storage.ComplianceOperatorProfileV2{storage.ComplianceOperatorProfileV2_builder{
					Name:           "ocp4",
					ProfileVersion: "test_version_ocp4",
					Title:          "test_title_ocp4",
				}.Build()}
				s.profilsDS.EXPECT().SearchProfiles(gomock.Any(), search.NewQueryBuilder().
					AddExactMatches(search.ComplianceOperatorScanRef, "test-ref").ProtoQuery()).Return(profilesOcp, nil).AnyTimes()
				s.benchmarkDS.EXPECT().GetBenchmarksByProfileName(gomock.Any(), "ocp4").Return(fixtures.GetExpectedBenchmark(), nil).AnyTimes()
				s.ruleDS.EXPECT().GetControlsByRulesAndBenchmarks(gomock.Any(), []string{"rule-name"}, []string{"OCP_CIS"}).Return(getExpectedControlResults(), nil).AnyTimes()
			},
		},
		{
			desc: "Query with custom pagination",
			query: apiV2.RawQuery_builder{
				Query:      "",
				Pagination: apiV2.Pagination_builder{Limit: 1}.Build(),
			}.Build(),
			expectedErr: nil,
			found:       true,
			setMocks: func() {
				expectedQ := search.EmptyQuery()
				countQuery := expectedQ.CloneVT()
				expectedQ.SetPagination(v1.QueryPagination_builder{Limit: 1}.Build())
				returnResults := []*storage.ComplianceOperatorCheckResultV2{
					convertUtils.GetComplianceStorageResults(s.T())[0],
				}

				s.resultDatastore.EXPECT().SearchComplianceCheckResults(gomock.Any(), expectedQ).Return(returnResults, nil).Times(1)
				s.ruleDS.EXPECT().SearchRules(gomock.Any(), gomock.Any()).Return([]*storage.ComplianceOperatorRuleV2{storage.ComplianceOperatorRuleV2_builder{Name: "rule-name"}.Build()}, nil).AnyTimes()
				s.resultDatastore.EXPECT().CountCheckResults(gomock.Any(), countQuery).Return(7, nil).Times(1)

				profilesOcp := []*storage.ComplianceOperatorProfileV2{storage.ComplianceOperatorProfileV2_builder{
					Name:           "ocp4",
					ProfileVersion: "test_version_ocp4",
					Title:          "test_title_ocp4",
				}.Build()}
				s.profilsDS.EXPECT().SearchProfiles(gomock.Any(), search.NewQueryBuilder().
					AddExactMatches(search.ComplianceOperatorScanRef, "test-ref").ProtoQuery()).Return(profilesOcp, nil).AnyTimes()
				s.benchmarkDS.EXPECT().GetBenchmarksByProfileName(gomock.Any(), "ocp4").Return(fixtures.GetExpectedBenchmark(), nil).AnyTimes()
				s.ruleDS.EXPECT().GetControlsByRulesAndBenchmarks(gomock.Any(), []string{"rule-name"}, []string{"OCP_CIS"}).Return(getExpectedControlResults(), nil).AnyTimes()
			},
		},
		{
			desc:        "Query with non-existent field",
			query:       apiV2.RawQuery_builder{Query: "Cluster ID:id"}.Build(),
			expectedErr: errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance scan results for query %v", apiV2.RawQuery_builder{Query: "Cluster ID:id"}.Build()),
			found:       false,
			setMocks: func() {
				expectedQ := search.NewQueryBuilder().AddStrings(search.ClusterID, "id").ProtoQuery()
				countQuery := expectedQ.CloneVT()
				expectedQ.SetPagination(v1.QueryPagination_builder{Limit: maxPaginationLimit}.Build())

				s.resultDatastore.EXPECT().SearchComplianceCheckResults(gomock.Any(), expectedQ).Return(nil, nil).Times(1)
				s.ruleDS.EXPECT().SearchRules(gomock.Any(), gomock.Any()).Return([]*storage.ComplianceOperatorRuleV2{storage.ComplianceOperatorRuleV2_builder{Name: "rule-name"}.Build()}, nil).AnyTimes()
				s.resultDatastore.EXPECT().CountCheckResults(gomock.Any(), countQuery).Return(7, nil).Times(1)
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			tc.setMocks()

			results, err := s.service.GetComplianceScanResults(s.ctx, tc.query)
			if tc.expectedErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(tc.expectedErr, err)
			}

			if tc.expectedResp != nil {
				protoassert.SlicesEqual(s.T(), convertUtils.GetConvertedComplianceData(s.T()), results.GetScanResults())
			}
		})
	}
}

func (s *ComplianceResultsServiceTestSuite) TestGetComplianceScanResult() {
	testCases := []struct {
		desc         string
		query        *apiV2.ResourceByID
		expectedResp *apiV2.ComplianceClusterCheckStatus
		expectedErr  error
		found        bool
		setMocks     func()
	}{
		{
			desc:         "ID exists",
			query:        apiV2.ResourceByID_builder{Id: uuid.NewDummy().String()}.Build(),
			expectedErr:  nil,
			expectedResp: convertUtils.GetConvertedComplianceResult(s.T(), scan1.GetLastExecutedTime()),
			found:        true,
			setMocks: func() {
				checkResult := convertUtils.GetComplianceStorageResult(s.T())
				s.resultDatastore.EXPECT().GetComplianceCheckResult(gomock.Any(), uuid.NewDummy().String()).Return(checkResult, true, nil).Times(1)

				scanQuery := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorScanRef, checkResult.GetScanRefId()).ProtoQuery()
				s.scanDS.EXPECT().SearchScans(gomock.Any(), scanQuery).Return([]*storage.ComplianceOperatorScanV2{scan1}, nil).Times(1)

				profilesOcp := []*storage.ComplianceOperatorProfileV2{storage.ComplianceOperatorProfileV2_builder{
					Name:           "ocp4",
					ProfileVersion: "test_version_ocp4",
					Title:          "test_title_ocp4",
				}.Build()}
				s.profilsDS.EXPECT().SearchProfiles(gomock.Any(), search.NewQueryBuilder().
					AddExactMatches(search.ComplianceOperatorScanRef, "test-ref").ProtoQuery()).Return(profilesOcp, nil).Times(1)
				ruleQuery := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorRuleRef, "test-ref-id").ProtoQuery()
				s.ruleDS.EXPECT().SearchRules(gomock.Any(), ruleQuery).Return([]*storage.ComplianceOperatorRuleV2{storage.ComplianceOperatorRuleV2_builder{Name: "rule-name"}.Build()}, nil).Times(1)
				s.benchmarkDS.EXPECT().GetBenchmarksByProfileName(gomock.Any(), "ocp4").Return(fixtures.GetExpectedBenchmark(), nil).Times(1)
				s.ruleDS.EXPECT().GetControlsByRulesAndBenchmarks(gomock.Any(), []string{"rule-name"}, []string{"OCP_CIS"}).Return(getExpectedControlResults(), nil).Times(1)
			},
		},
		{
			desc:        "Query with non-existent record",
			query:       apiV2.ResourceByID_builder{Id: uuid.NewDummy().String()}.Build(),
			expectedErr: errors.Wrapf(errox.InvalidArgs, "compliance check result with id %q does not exist", uuid.NewDummy().String()),
			found:       false,
			setMocks: func() {
				s.resultDatastore.EXPECT().GetComplianceCheckResult(gomock.Any(), uuid.NewDummy().String()).Return(nil, false, nil).Times(1)
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			tc.setMocks()

			result, err := s.service.GetComplianceScanCheckResult(s.ctx, tc.query)
			if tc.expectedErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(tc.expectedErr, err)
			}

			if tc.expectedResp != nil {
				protoassert.Equal(s.T(), convertUtils.GetConvertedComplianceResult(s.T(), scan1.GetLastExecutedTime()), result)
			}
		})
	}
}

func (s *ComplianceResultsServiceTestSuite) TestGetComplianceScanConfigurationResults() {
	testCases := []struct {
		desc         string
		query        *apiV2.ComplianceScanResultsRequest
		expectedResp []*apiV2.ComplianceCheckData
		expectedErr  error
		found        bool
		setMocks     func()
	}{
		{
			desc: "Empty query",
			query: apiV2.ComplianceScanResultsRequest_builder{
				ScanConfigName: "scanConfig1",
				Query:          apiV2.RawQuery_builder{Query: ""}.Build(),
			}.Build(),
			expectedErr:  nil,
			expectedResp: convertUtils.GetConvertedComplianceData(s.T()),
			found:        true,
			setMocks: func() {
				expectedQ := search.ConjunctionQuery(
					search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorScanConfigName, "scanConfig1").ProtoQuery(),
					search.EmptyQuery(),
				)
				countQuery := expectedQ.CloneVT()
				expectedQ.SetPagination(v1.QueryPagination_builder{Limit: maxPaginationLimit}.Build())

				s.resultDatastore.EXPECT().SearchComplianceCheckResults(gomock.Any(), expectedQ).Return(convertUtils.GetComplianceStorageResults(s.T()), nil).Times(1)
				s.ruleDS.EXPECT().SearchRules(gomock.Any(), gomock.Any()).Return([]*storage.ComplianceOperatorRuleV2{storage.ComplianceOperatorRuleV2_builder{Name: "rule-name"}.Build()}, nil).AnyTimes()
				s.resultDatastore.EXPECT().CountCheckResults(gomock.Any(), countQuery).Return(7, nil).Times(1)
				profilesOcp := []*storage.ComplianceOperatorProfileV2{storage.ComplianceOperatorProfileV2_builder{
					Name:           "ocp4",
					ProfileVersion: "test_version_ocp4",
					Title:          "test_title_ocp4",
				}.Build()}
				s.profilsDS.EXPECT().SearchProfiles(gomock.Any(), search.NewQueryBuilder().
					AddExactMatches(search.ComplianceOperatorScanRef, "test-ref").ProtoQuery()).Return(profilesOcp, nil).AnyTimes()
				s.benchmarkDS.EXPECT().GetBenchmarksByProfileName(gomock.Any(), "ocp4").Return(fixtures.GetExpectedBenchmark(), nil).AnyTimes()
				s.ruleDS.EXPECT().GetControlsByRulesAndBenchmarks(gomock.Any(), []string{"rule-name"}, []string{"OCP_CIS"}).Return(getExpectedControlResults(), nil).AnyTimes()
			},
		},
		{
			desc: "Query with search field",
			query: apiV2.ComplianceScanResultsRequest_builder{
				ScanConfigName: "scanConfig1",
				Query:          apiV2.RawQuery_builder{Query: "Cluster ID:" + fixtureconsts.Cluster1}.Build(),
			}.Build(),
			expectedErr: nil,
			found:       true,
			setMocks: func() {
				expectedQ := search.NewQueryBuilder().AddStrings(search.ClusterID, fixtureconsts.Cluster1).ProtoQuery()
				expectedQ = search.ConjunctionQuery(
					search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorScanConfigName, "scanConfig1").ProtoQuery(),
					expectedQ,
				)

				countQuery := expectedQ.CloneVT()
				expectedQ.SetPagination(v1.QueryPagination_builder{Limit: maxPaginationLimit}.Build())

				s.resultDatastore.EXPECT().SearchComplianceCheckResults(gomock.Any(), expectedQ).Return(convertUtils.GetOneClusterComplianceStorageResults(s.T(), fixtureconsts.Cluster1), nil).Times(1)
				s.ruleDS.EXPECT().SearchRules(gomock.Any(), gomock.Any()).Return([]*storage.ComplianceOperatorRuleV2{storage.ComplianceOperatorRuleV2_builder{Name: "rule-name"}.Build()}, nil).AnyTimes()
				s.resultDatastore.EXPECT().CountCheckResults(gomock.Any(), countQuery).Return(7, nil).Times(1)
			},
		},
		{
			desc: "Query with custom pagination",
			query: apiV2.ComplianceScanResultsRequest_builder{
				ScanConfigName: "scanConfig1",
				Query: apiV2.RawQuery_builder{Query: "",
					Pagination: apiV2.Pagination_builder{Limit: 1}.Build()}.Build(),
			}.Build(),
			expectedErr: nil,
			found:       true,
			setMocks: func() {
				expectedQ := search.ConjunctionQuery(
					search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorScanConfigName, "scanConfig1").ProtoQuery(),
					search.EmptyQuery(),
				)
				countQuery := expectedQ.CloneVT()
				expectedQ.SetPagination(v1.QueryPagination_builder{Limit: 1}.Build())
				returnResults := []*storage.ComplianceOperatorCheckResultV2{
					convertUtils.GetComplianceStorageResults(s.T())[0],
				}

				s.resultDatastore.EXPECT().SearchComplianceCheckResults(gomock.Any(), expectedQ).Return(returnResults, nil).Times(1)
				s.ruleDS.EXPECT().SearchRules(gomock.Any(), gomock.Any()).Return([]*storage.ComplianceOperatorRuleV2{storage.ComplianceOperatorRuleV2_builder{Name: "rule-name"}.Build()}, nil).AnyTimes()
				s.resultDatastore.EXPECT().CountCheckResults(gomock.Any(), countQuery).Return(7, nil).Times(1)
			},
		},
		{
			desc: "Query with no scan configuration name field",
			query: apiV2.ComplianceScanResultsRequest_builder{
				Query: apiV2.RawQuery_builder{Query: "Cluster ID:" + fixtureconsts.Cluster1}.Build(),
			}.Build(),
			expectedErr: errors.Wrap(errox.InvalidArgs, "Scan configuration name is required"),
			found:       false,
			setMocks:    func() {},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			tc.setMocks()

			results, err := s.service.GetComplianceScanConfigurationResults(s.ctx, tc.query)
			if tc.expectedErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(tc.expectedErr, err)
			}

			if tc.expectedResp != nil {
				protoassert.ElementsMatch(s.T(), tc.expectedResp, results.GetScanResults())
			}
		})
	}
}

func (s *ComplianceResultsServiceTestSuite) TestGetComplianceProfileResults() {
	testCases := []struct {
		desc         string
		query        *apiV2.ComplianceProfileResultsRequest
		expectedResp *apiV2.ListComplianceProfileResults
		expectedErr  error
		setMocks     func()
	}{
		{
			desc: "Empty query",
			query: apiV2.ComplianceProfileResultsRequest_builder{
				ProfileName: "ocp4",
				Query:       apiV2.RawQuery_builder{Query: ""}.Build(),
			}.Build(),
			expectedErr:  nil,
			expectedResp: convertUtils.GetComplianceProfileResultsV2(s.T(), "ocp4"),
			setMocks: func() {
				expectedQ := search.ConjunctionQuery(
					search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, "ocp4").ProtoQuery(),
					search.EmptyQuery(),
				)
				countQuery := expectedQ.CloneVT()
				expectedQ.SetPagination(v1.QueryPagination_builder{Limit: maxPaginationLimit}.Build())

				results := []*datastore.ResourceResultsByProfile{
					convertUtils.GetComplianceStorageProfileResults(s.T(), "ocp4"),
				}
				s.resultDatastore.EXPECT().CountByField(gomock.Any(), countQuery, search.ComplianceOperatorCheckName).Return(1, nil).Times(1)
				s.resultDatastore.EXPECT().ComplianceProfileResults(gomock.Any(), expectedQ).Return(results, nil).Times(1)
				s.benchmarkDS.EXPECT().GetBenchmarksByProfileName(gomock.Any(), "ocp4").Return(fixtures.GetExpectedBenchmark(), nil).Times(1)
				s.ruleDS.EXPECT().GetControlsByRulesAndBenchmarks(gomock.Any(), []string{"rule-name"}, []string{"OCP_CIS"}).Return(getExpectedControlResults(), nil).Times(1)
			},
		},
		{
			desc: "Query with search field",
			query: apiV2.ComplianceProfileResultsRequest_builder{
				ProfileName: "ocp4",
				Query:       apiV2.RawQuery_builder{Query: "Cluster ID:" + fixtureconsts.Cluster1}.Build(),
			}.Build(),
			expectedErr:  nil,
			expectedResp: convertUtils.GetComplianceProfileResultsV2(s.T(), "ocp4"),
			setMocks: func() {
				expectedQ := search.NewQueryBuilder().AddStrings(search.ClusterID, fixtureconsts.Cluster1).ProtoQuery()
				expectedQ = search.ConjunctionQuery(
					search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, "ocp4").ProtoQuery(),
					expectedQ,
				)
				countQuery := expectedQ.CloneVT()
				expectedQ.SetPagination(v1.QueryPagination_builder{Limit: maxPaginationLimit}.Build())

				results := []*datastore.ResourceResultsByProfile{
					convertUtils.GetComplianceStorageProfileResults(s.T(), "ocp4"),
				}
				s.resultDatastore.EXPECT().CountByField(gomock.Any(), countQuery, search.ComplianceOperatorCheckName).Return(1, nil).Times(1)
				s.resultDatastore.EXPECT().ComplianceProfileResults(gomock.Any(), expectedQ).Return(results, nil).Times(1)

				s.benchmarkDS.EXPECT().GetBenchmarksByProfileName(gomock.Any(), "ocp4").Return(fixtures.GetExpectedBenchmark(), nil).Times(1)
				s.ruleDS.EXPECT().GetControlsByRulesAndBenchmarks(gomock.Any(), []string{"rule-name"}, []string{"OCP_CIS"}).Return(getExpectedControlResults(), nil).Times(1)
			},
		},
		{
			desc: "Query with non-existent field",
			query: apiV2.ComplianceProfileResultsRequest_builder{
				ProfileName: "",
				Query:       apiV2.RawQuery_builder{Query: "Cluster ID:" + fixtureconsts.Cluster1}.Build(),
			}.Build(),
			expectedErr: errors.Wrap(errox.InvalidArgs, "Profile name is required"),
			setMocks: func() {
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			tc.setMocks()

			results, err := s.service.GetComplianceProfileResults(s.ctx, tc.query)
			if tc.expectedErr == nil {
				s.Require().NoError(err)
				protoassert.Equal(s.T(), tc.expectedResp, results)
			} else {
				s.Require().Error(tc.expectedErr, err)
				s.Require().Nil(results)
			}
		})
	}
}

func getExpectedControlResults() []*ruleDS.ControlResult {
	return []*ruleDS.ControlResult{
		{RuleName: "rule-name", Standard: "OCP-CIS", Control: "1.2.2"},
		{RuleName: "rule-name", Standard: "OCP-CIS", Control: "1.3.3"},
		{RuleName: "rule-name", Standard: "OCP-CIS", Control: "1.4.4"},
	}
}

func (s *ComplianceResultsServiceTestSuite) TestGetComplianceProfileCheckResult() {
	testCases := []struct {
		desc         string
		query        *apiV2.ComplianceProfileCheckRequest
		expectedResp *apiV2.ListComplianceCheckClusterResponse
		expectedErr  error
		setMocks     func()
	}{
		{
			desc: "Empty query",
			query: apiV2.ComplianceProfileCheckRequest_builder{
				ProfileName: "ocp4",
				CheckName:   "check-name",
				Query:       apiV2.RawQuery_builder{Query: ""}.Build(),
			}.Build(),
			expectedErr: nil,
			expectedResp: apiV2.ListComplianceCheckClusterResponse_builder{
				CheckResults: convertUtils.GetConvertedComplianceResult(s.T(), scan1.GetLastExecutedTime()).GetClusters(),
				ProfileName:  "ocp4",
				CheckName:    "check-name",
				TotalCount:   7,
				Controls: []*apiV2.ComplianceControl{
					apiV2.ComplianceControl_builder{Standard: "OCP-CIS", Control: "1.2.2"}.Build(),
					apiV2.ComplianceControl_builder{Standard: "OCP-CIS", Control: "1.3.3"}.Build(),
					apiV2.ComplianceControl_builder{Standard: "OCP-CIS", Control: "1.4.4"}.Build(),
				},
			}.Build(),
			setMocks: func() {
				expectedQ := search.ConjunctionQuery(
					search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, "ocp4").
						AddExactMatches(search.ComplianceOperatorCheckName, "check-name").ProtoQuery(),
					search.EmptyQuery(),
				)
				countQuery := expectedQ.CloneVT()
				expectedQ.SetPagination(v1.QueryPagination_builder{Limit: maxPaginationLimit}.Build())

				s.resultDatastore.EXPECT().SearchComplianceCheckResults(gomock.Any(), expectedQ).
					Return([]*storage.ComplianceOperatorCheckResultV2{convertUtils.GetComplianceStorageResult(s.T())}, nil).
					Times(1)
				s.resultDatastore.EXPECT().CountCheckResults(gomock.Any(), countQuery).Return(7, nil).Times(1)

				scanQuery := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, "ocp4").AddExactMatches(search.ClusterID, fixtureconsts.Cluster1).ProtoQuery()
				s.scanDS.EXPECT().SearchScans(gomock.Any(), scanQuery).Return([]*storage.ComplianceOperatorScanV2{scan1}, nil).Times(1)

				ruleQuery := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorRuleRef, "test-ref-id").ProtoQuery()
				s.ruleDS.EXPECT().SearchRules(gomock.Any(), ruleQuery).Return([]*storage.ComplianceOperatorRuleV2{storage.ComplianceOperatorRuleV2_builder{Name: "rule-name"}.Build()}, nil).Times(1)
				s.benchmarkDS.EXPECT().GetBenchmarksByProfileName(gomock.Any(), "ocp4").Return(fixtures.GetExpectedBenchmark(), nil).Times(1)
				s.ruleDS.EXPECT().GetControlsByRulesAndBenchmarks(gomock.Any(), []string{"rule-name"}, []string{"OCP_CIS"}).Return(getExpectedControlResults(), nil).Times(1)
			},
		},
		{
			desc: "Query with search field",
			query: apiV2.ComplianceProfileCheckRequest_builder{
				ProfileName: "ocp4",
				CheckName:   "check-name",
				Query:       apiV2.RawQuery_builder{Query: "Cluster ID:" + fixtureconsts.Cluster1}.Build(),
			}.Build(),
			expectedErr: nil,
			expectedResp: apiV2.ListComplianceCheckClusterResponse_builder{
				CheckResults: convertUtils.GetConvertedComplianceResult(s.T(), scan1.GetLastExecutedTime()).GetClusters(),
				ProfileName:  "ocp4",
				CheckName:    "check-name",
				TotalCount:   3,
				Controls: []*apiV2.ComplianceControl{
					apiV2.ComplianceControl_builder{Standard: "OCP-CIS", Control: "1.2.2"}.Build(),
					apiV2.ComplianceControl_builder{Standard: "OCP-CIS", Control: "1.3.3"}.Build(),
					apiV2.ComplianceControl_builder{Standard: "OCP-CIS", Control: "1.4.4"}.Build(),
				},
			}.Build(),
			setMocks: func() {
				expectedQ := search.NewQueryBuilder().AddStrings(search.ClusterID, fixtureconsts.Cluster1).ProtoQuery()
				expectedQ = search.ConjunctionQuery(
					search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, "ocp4").
						AddExactMatches(search.ComplianceOperatorCheckName, "check-name").ProtoQuery(),
					expectedQ,
				)
				countQuery := expectedQ.CloneVT()
				expectedQ.SetPagination(v1.QueryPagination_builder{Limit: maxPaginationLimit}.Build())

				s.resultDatastore.EXPECT().SearchComplianceCheckResults(gomock.Any(), expectedQ).
					Return([]*storage.ComplianceOperatorCheckResultV2{convertUtils.GetComplianceStorageResult(s.T())}, nil).
					Times(1)
				s.resultDatastore.EXPECT().CountCheckResults(gomock.Any(), countQuery).Return(3, nil).Times(1)

				scanQuery := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, "ocp4").AddExactMatches(search.ClusterID, fixtureconsts.Cluster1).ProtoQuery()
				s.scanDS.EXPECT().SearchScans(gomock.Any(), scanQuery).Return([]*storage.ComplianceOperatorScanV2{scan1}, nil).Times(1)

				ruleQuery := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorRuleRef, "test-ref-id").ProtoQuery()
				s.ruleDS.EXPECT().SearchRules(gomock.Any(), ruleQuery).Return([]*storage.ComplianceOperatorRuleV2{storage.ComplianceOperatorRuleV2_builder{Name: "rule-name"}.Build()}, nil).Times(1)
				s.benchmarkDS.EXPECT().GetBenchmarksByProfileName(gomock.Any(), "ocp4").Return(fixtures.GetExpectedBenchmark(), nil).Times(1)
				s.ruleDS.EXPECT().GetControlsByRulesAndBenchmarks(gomock.Any(), []string{"rule-name"}, []string{"OCP_CIS"}).Return(getExpectedControlResults(), nil).Times(1)
			},
		},
		{
			desc: "Query with non-existent field",
			query: apiV2.ComplianceProfileCheckRequest_builder{
				ProfileName: "",
				Query:       apiV2.RawQuery_builder{Query: "Cluster ID:" + fixtureconsts.Cluster1}.Build(),
			}.Build(),
			expectedErr: errors.Wrap(errox.InvalidArgs, "Profile name is required"),
			setMocks: func() {
			},
		},
		{
			desc: "Query with non-existent field",
			query: apiV2.ComplianceProfileCheckRequest_builder{
				ProfileName: "ocp4",
				Query:       apiV2.RawQuery_builder{Query: "Cluster ID:" + fixtureconsts.Cluster1}.Build(),
			}.Build(),
			expectedErr: errors.Wrap(errox.InvalidArgs, "Compliance check name is required"),
			setMocks: func() {
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			tc.setMocks()

			results, err := s.service.GetComplianceProfileCheckResult(s.ctx, tc.query)
			if tc.expectedErr == nil {
				s.Require().NoError(err)
				protoassert.Equal(s.T(), tc.expectedResp, results)
			} else {
				s.Require().Error(tc.expectedErr, err)
				s.Require().Nil(results)
			}
		})
	}
}

func (s *ComplianceResultsServiceTestSuite) TestGetComplianceProfileClusterResults() {
	testCases := []struct {
		desc         string
		query        *apiV2.ComplianceProfileClusterRequest
		expectedResp *apiV2.ListComplianceCheckResultResponse
		expectedErr  error
		setMocks     func()
	}{
		{
			desc: "Empty query",
			query: apiV2.ComplianceProfileClusterRequest_builder{
				ProfileName: "ocp4",
				ClusterId:   testconsts.Cluster1,
				Query:       apiV2.RawQuery_builder{Query: ""}.Build(),
			}.Build(),
			expectedErr: nil,
			expectedResp: apiV2.ListComplianceCheckResultResponse_builder{
				CheckResults: convertUtils.GetConvertedCheckResult(s.T()),
				ProfileName:  "ocp4",
				ClusterId:    testconsts.Cluster1,
				TotalCount:   7,
				LastScanTime: scan1.GetLastExecutedTime(),
			}.Build(),
			setMocks: func() {
				expectedQ := search.ConjunctionQuery(
					search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, "ocp4").
						AddExactMatches(search.ClusterID, testconsts.Cluster1).ProtoQuery(),
					search.EmptyQuery(),
				)
				countQuery := expectedQ.CloneVT()
				expectedQ.SetPagination(v1.QueryPagination_builder{Limit: maxPaginationLimit}.Build())

				ruleQuery := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorRuleRef, "test-ref-id").ProtoQuery()
				s.ruleDS.EXPECT().SearchRules(gomock.Any(), ruleQuery).Return([]*storage.ComplianceOperatorRuleV2{storage.ComplianceOperatorRuleV2_builder{Name: "rule-name"}.Build()}, nil).Times(1)

				s.resultDatastore.EXPECT().SearchComplianceCheckResults(gomock.Any(), expectedQ).
					Return([]*storage.ComplianceOperatorCheckResultV2{convertUtils.GetComplianceStorageResult(s.T())}, nil).
					Times(1)
				s.resultDatastore.EXPECT().CountCheckResults(gomock.Any(), countQuery).Return(7, nil).Times(1)

				scanQuery := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, "ocp4").AddExactMatches(search.ClusterID, testconsts.Cluster1).ProtoQuery()
				s.scanDS.EXPECT().SearchScans(gomock.Any(), scanQuery).Return([]*storage.ComplianceOperatorScanV2{scan1}, nil).Times(1)

				s.benchmarkDS.EXPECT().GetBenchmarksByProfileName(gomock.Any(), "ocp4").Return(fixtures.GetExpectedBenchmark(), nil).Times(1)
				s.ruleDS.EXPECT().GetControlsByRulesAndBenchmarks(gomock.Any(), []string{"rule-name"}, []string{"OCP_CIS"}).Return(getExpectedControlResults(), nil).Times(1)
			},
		},
		{
			desc: "Query with search field",
			query: apiV2.ComplianceProfileClusterRequest_builder{
				ProfileName: "ocp4",
				ClusterId:   testconsts.Cluster1,
				Query:       apiV2.RawQuery_builder{Query: "Compliance Check Name:" + "check-name"}.Build(),
			}.Build(),
			expectedErr: nil,
			expectedResp: apiV2.ListComplianceCheckResultResponse_builder{
				CheckResults: convertUtils.GetConvertedCheckResult(s.T()),
				ProfileName:  "ocp4",
				ClusterId:    testconsts.Cluster1,
				TotalCount:   3,
				LastScanTime: scan1.GetLastExecutedTime(),
			}.Build(),
			setMocks: func() {
				expectedQ := search.NewQueryBuilder().AddStrings(search.ComplianceOperatorCheckName, "check-name").ProtoQuery()
				expectedQ = search.ConjunctionQuery(
					search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, "ocp4").
						AddExactMatches(search.ClusterID, testconsts.Cluster1).ProtoQuery(),
					expectedQ,
				)
				countQuery := expectedQ.CloneVT()
				expectedQ.SetPagination(v1.QueryPagination_builder{Limit: maxPaginationLimit}.Build())

				ruleQuery := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorRuleRef, "test-ref-id").ProtoQuery()
				s.ruleDS.EXPECT().SearchRules(gomock.Any(), ruleQuery).Return([]*storage.ComplianceOperatorRuleV2{storage.ComplianceOperatorRuleV2_builder{Name: "rule-name"}.Build()}, nil).Times(1)

				s.resultDatastore.EXPECT().SearchComplianceCheckResults(gomock.Any(), expectedQ).
					Return([]*storage.ComplianceOperatorCheckResultV2{convertUtils.GetComplianceStorageResult(s.T())}, nil).
					Times(1)
				s.resultDatastore.EXPECT().CountCheckResults(gomock.Any(), countQuery).Return(3, nil).Times(1)

				scanQuery := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, "ocp4").AddExactMatches(search.ClusterID, testconsts.Cluster1).ProtoQuery()
				s.scanDS.EXPECT().SearchScans(gomock.Any(), scanQuery).Return([]*storage.ComplianceOperatorScanV2{scan1}, nil).Times(1)

				s.benchmarkDS.EXPECT().GetBenchmarksByProfileName(gomock.Any(), "ocp4").Return(fixtures.GetExpectedBenchmark(), nil).Times(1)
				s.ruleDS.EXPECT().GetControlsByRulesAndBenchmarks(gomock.Any(), []string{"rule-name"}, []string{"OCP_CIS"}).Return(getExpectedControlResults(), nil).Times(1)
			},
		},
		{
			desc: "Request with missing field",
			query: apiV2.ComplianceProfileClusterRequest_builder{
				ProfileName: "",
				Query:       apiV2.RawQuery_builder{Query: "Cluster ID:" + testconsts.Cluster1}.Build(),
			}.Build(),
			expectedErr: errors.Wrap(errox.InvalidArgs, "Profile name is required"),
			setMocks: func() {
			},
		},
		{
			desc: "Query with missing cluster",
			query: apiV2.ComplianceProfileClusterRequest_builder{
				ProfileName: "ocp4",
				Query:       apiV2.RawQuery_builder{Query: "Compliance Operator Check Name:" + "check-name"}.Build(),
			}.Build(),
			expectedErr: errors.Wrap(errox.InvalidArgs, "Cluster ID is required"),
			setMocks: func() {
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			tc.setMocks()

			results, err := s.service.GetComplianceProfileClusterResults(s.ctx, tc.query)
			if tc.expectedErr == nil {
				s.Require().NoError(err)
				protoassert.Equal(s.T(), tc.expectedResp, results)
			} else {
				s.Require().Error(tc.expectedErr, err)
				s.Require().Nil(results)
			}
		})
	}
}

func (s *ComplianceResultsServiceTestSuite) TestGetComplianceProfileCheckDetails() {
	testCases := []struct {
		desc         string
		query        *apiV2.ComplianceCheckDetailRequest
		expectedResp *apiV2.ComplianceClusterCheckStatus
		expectedErr  error
		found        bool
		setMocks     func()
	}{
		{
			desc: "check exists",
			query: apiV2.ComplianceCheckDetailRequest_builder{
				ProfileName: "ocp-4",
				CheckName:   "check1",
			}.Build(),
			expectedErr:  nil,
			expectedResp: convertUtils.GetConvertedComplianceResult(s.T(), scan1.GetLastExecutedTime()),
			found:        true,
			setMocks: func() {
				expectedQ := search.ConjunctionQuery(
					search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, "ocp-4").
						AddExactMatches(search.ComplianceOperatorCheckName, "check1").ProtoQuery(),
					search.EmptyQuery(),
				)

				checkResult := convertUtils.GetComplianceStorageResult(s.T())
				s.resultDatastore.EXPECT().SearchComplianceCheckResults(gomock.Any(), expectedQ).Return([]*storage.ComplianceOperatorCheckResultV2{checkResult}, nil).Times(1)

				s.ruleDS.EXPECT().SearchRules(gomock.Any(), gomock.Any()).Return([]*storage.ComplianceOperatorRuleV2{storage.ComplianceOperatorRuleV2_builder{Name: "rule-name"}.Build()}, nil).AnyTimes()

				profilesOcp := []*storage.ComplianceOperatorProfileV2{storage.ComplianceOperatorProfileV2_builder{
					Name:           "ocp4",
					ProfileVersion: "test_version_ocp4",
					Title:          "test_title_ocp4",
				}.Build()}
				s.profilsDS.EXPECT().SearchProfiles(gomock.Any(), search.NewQueryBuilder().
					AddExactMatches(search.ComplianceOperatorScanRef, "test-ref").ProtoQuery()).Return(profilesOcp, nil).Times(1)
				s.benchmarkDS.EXPECT().GetBenchmarksByProfileName(gomock.Any(), "ocp4").Return(fixtures.GetExpectedBenchmark(), nil).Times(1)
				s.ruleDS.EXPECT().GetControlsByRulesAndBenchmarks(gomock.Any(), []string{"rule-name"}, []string{"OCP_CIS"}).Return(getExpectedControlResults(), nil).Times(1)
			},
		},
		{
			desc: "check exists - cluster query",
			query: apiV2.ComplianceCheckDetailRequest_builder{
				ProfileName: "ocp-4",
				CheckName:   "check1",
				Query:       apiV2.RawQuery_builder{Query: "Cluster ID:" + testconsts.Cluster1}.Build(),
			}.Build(),
			expectedErr:  nil,
			expectedResp: convertUtils.GetConvertedComplianceResult(s.T(), scan1.GetLastExecutedTime()),
			found:        true,
			setMocks: func() {
				expectedQ := search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster1).ProtoQuery()
				expectedQ = search.ConjunctionQuery(
					search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, "ocp-4").
						AddExactMatches(search.ComplianceOperatorCheckName, "check1").ProtoQuery(),
					expectedQ,
				)

				checkResult := convertUtils.GetComplianceStorageResult(s.T())
				s.resultDatastore.EXPECT().SearchComplianceCheckResults(gomock.Any(), expectedQ).Return([]*storage.ComplianceOperatorCheckResultV2{checkResult}, nil).Times(1)

				s.ruleDS.EXPECT().SearchRules(gomock.Any(), gomock.Any()).Return([]*storage.ComplianceOperatorRuleV2{storage.ComplianceOperatorRuleV2_builder{Name: "rule-name"}.Build()}, nil).AnyTimes()

				profilesOcp := []*storage.ComplianceOperatorProfileV2{storage.ComplianceOperatorProfileV2_builder{
					Name:           "ocp4",
					ProfileVersion: "test_version_ocp4",
					Title:          "test_title_ocp4",
				}.Build()}
				s.profilsDS.EXPECT().SearchProfiles(gomock.Any(), search.NewQueryBuilder().
					AddExactMatches(search.ComplianceOperatorScanRef, "test-ref").ProtoQuery()).Return(profilesOcp, nil).Times(1)
				s.benchmarkDS.EXPECT().GetBenchmarksByProfileName(gomock.Any(), "ocp4").Return(fixtures.GetExpectedBenchmark(), nil).Times(1)
				s.ruleDS.EXPECT().GetControlsByRulesAndBenchmarks(gomock.Any(), []string{"rule-name"}, []string{"OCP_CIS"}).Return(getExpectedControlResults(), nil).Times(1)
			},
		},
		{
			desc: "Query with non-existent record",
			query: apiV2.ComplianceCheckDetailRequest_builder{
				ProfileName: "ocp-4",
				CheckName:   "check1",
			}.Build(),
			expectedErr: errors.Wrapf(errox.InvalidArgs, "compliance check result with id %q does not exist", uuid.NewDummy().String()),
			found:       false,
			setMocks: func() {
				expectedQ := search.ConjunctionQuery(
					search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, "ocp-4").
						AddExactMatches(search.ComplianceOperatorCheckName, "check1").ProtoQuery(),
					search.EmptyQuery(),
				)

				s.resultDatastore.EXPECT().SearchComplianceCheckResults(gomock.Any(), expectedQ).Return(nil, nil).Times(1)
			},
		},
		{
			desc: "bad request -- no profile",
			query: apiV2.ComplianceCheckDetailRequest_builder{
				CheckName: "check1",
			}.Build(),
			expectedErr: errors.Wrap(errox.InvalidArgs, "Profile name is required"),
			found:       false,
			setMocks: func() {
			},
		},
		{
			desc: "bad request -- no check",
			query: apiV2.ComplianceCheckDetailRequest_builder{
				ProfileName: "profile-name",
			}.Build(),
			expectedErr: errors.Wrap(errox.InvalidArgs, "Check name is required"),
			found:       false,
			setMocks: func() {
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			tc.setMocks()

			result, err := s.service.GetComplianceProfileCheckDetails(s.ctx, tc.query)
			if tc.expectedErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(tc.expectedErr, err)
			}

			if tc.expectedResp != nil {
				protoassert.Equal(s.T(), convertUtils.GetConvertedComplianceResult(s.T(), nil), result)
			}
		})
	}
}
