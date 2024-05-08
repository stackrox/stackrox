package service

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	resultMocks "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore/mocks"
	integrationMocks "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore/mocks"
	profileDatastore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore/mocks"
	scanConfigMocks "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore/mocks"
	scanMocks "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore/mocks"
	convertUtils "github.com/stackrox/rox/central/convert/testutils"
	v1 "github.com/stackrox/rox/generated/api/v1"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	types "github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	integration1 = &storage.ComplianceIntegration{
		Id:                  "",
		ClusterId:           testconsts.Cluster1,
		ComplianceNamespace: fixtureconsts.Namespace1,
		Version:             "2",
		StatusErrors:        []string{"test error"},
	}

	integration2 = &storage.ComplianceIntegration{
		Id:                  "",
		ClusterId:           testconsts.Cluster2,
		ComplianceNamespace: fixtureconsts.Namespace1,
		Version:             "2",
		StatusErrors:        []string{"test error"},
	}

	integration3 = &storage.ComplianceIntegration{
		Id:                  "",
		ClusterId:           testconsts.Cluster3,
		ComplianceNamespace: fixtureconsts.Namespace1,
		Version:             "2",
		StatusErrors:        []string{"test error"},
	}

	scan1 = &storage.ComplianceOperatorScanV2{
		Id:               "",
		ClusterId:        testconsts.Cluster1,
		LastExecutedTime: types.TimestampNow(),
	}

	scan2 = &storage.ComplianceOperatorScanV2{
		Id:               "",
		ClusterId:        testconsts.Cluster2,
		LastExecutedTime: types.TimestampNow(),
	}

	scan3 = &storage.ComplianceOperatorScanV2{
		Id:               "",
		ClusterId:        testconsts.Cluster3,
		LastExecutedTime: types.TimestampNow(),
	}
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

func TestComplianceResultsStatsService(t *testing.T) {
	suite.Run(t, new(ComplianceResultsStatsServiceTestSuite))
}

type ComplianceResultsStatsServiceTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	ctx             context.Context
	resultDatastore *resultMocks.MockDataStore
	scanConfigDS    *scanConfigMocks.MockDataStore
	integrationDS   *integrationMocks.MockDataStore
	service         Service
	profileDS       *profileDatastore.MockDataStore
	scanDS          *scanMocks.MockDataStore
}

func (s *ComplianceResultsStatsServiceTestSuite) SetupSuite() {
	s.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")
	if !features.ComplianceEnhancements.Enabled() {
		s.T().Skip("Skip test when compliance enhancements are disabled")
		s.T().SkipNow()
	}

	s.ctx = sac.WithAllAccess(context.Background())
}

func (s *ComplianceResultsStatsServiceTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.resultDatastore = resultMocks.NewMockDataStore(s.mockCtrl)
	s.scanConfigDS = scanConfigMocks.NewMockDataStore(s.mockCtrl)
	s.integrationDS = integrationMocks.NewMockDataStore(s.mockCtrl)
	s.profileDS = profileDatastore.NewMockDataStore(s.mockCtrl)
	s.scanDS = scanMocks.NewMockDataStore(s.mockCtrl)

	s.service = New(s.resultDatastore, s.scanConfigDS, s.integrationDS, s.profileDS, s.scanDS)
}

func (s *ComplianceResultsStatsServiceTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ComplianceResultsStatsServiceTestSuite) TestGetComplianceClusterScanStats() {
	testCases := []struct {
		desc         string
		query        *apiV2.RawQuery
		expectedResp []*apiV2.ComplianceClusterScanStats
		expectedErr  error
		setMocks     func()
	}{
		{
			desc:        "Empty query",
			query:       &apiV2.RawQuery{Query: ""},
			expectedErr: nil,
			expectedResp: []*apiV2.ComplianceClusterScanStats{
				convertUtils.GetComplianceClusterScanV2Count(s.T(), fixtureconsts.Cluster1),
				convertUtils.GetComplianceClusterScanV2Count(s.T(), fixtureconsts.Cluster2),
				convertUtils.GetComplianceClusterScanV2Count(s.T(), fixtureconsts.Cluster3),
			},
			setMocks: func() {
				expectedQ := search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery()
				results := []*datastore.ResourceResultCountByClusterScan{
					convertUtils.GetComplianceStorageClusterScanCount(s.T(), fixtureconsts.Cluster1),
					convertUtils.GetComplianceStorageClusterScanCount(s.T(), fixtureconsts.Cluster2),
					convertUtils.GetComplianceStorageClusterScanCount(s.T(), fixtureconsts.Cluster3),
				}
				s.scanConfigDS.EXPECT().GetScanConfigurationByName(gomock.Any(), "scanConfig1").Return(getTestRec("scanConfig1"), nil).Times(1)
				s.resultDatastore.EXPECT().ComplianceCheckResultStats(gomock.Any(), expectedQ).Return(results, nil).Times(1)
			},
		},
		{
			desc:        "Query with search field",
			query:       &apiV2.RawQuery{Query: "Cluster ID:" + fixtureconsts.Cluster1},
			expectedErr: nil,
			expectedResp: []*apiV2.ComplianceClusterScanStats{
				convertUtils.GetComplianceClusterScanV2Count(s.T(), fixtureconsts.Cluster1),
			},
			setMocks: func() {
				expectedQ := search.NewQueryBuilder().AddStrings(search.ClusterID, fixtureconsts.Cluster1).
					WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery()

				results := []*datastore.ResourceResultCountByClusterScan{
					convertUtils.GetComplianceStorageClusterScanCount(s.T(), fixtureconsts.Cluster1),
				}
				s.scanConfigDS.EXPECT().GetScanConfigurationByName(gomock.Any(), "scanConfig1").Return(getTestRec("scanConfig1"), nil).Times(1)
				s.resultDatastore.EXPECT().ComplianceCheckResultStats(gomock.Any(), expectedQ).Return(results, nil).Times(1)
			},
		},
		{
			desc:        "Query with non-existent field",
			query:       &apiV2.RawQuery{Query: "Cluster ID:id"},
			expectedErr: errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance cluster scan stats for query %v", &apiV2.RawQuery{Query: "Cluster ID:id"}),
			setMocks: func() {
				expectedQ := search.NewQueryBuilder().AddStrings(search.ClusterID, "id").
					WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery()

				s.resultDatastore.EXPECT().ComplianceCheckResultStats(gomock.Any(), expectedQ).Return(nil, nil).Times(1)
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			tc.setMocks()

			results, err := s.service.GetComplianceClusterScanStats(s.ctx, tc.query)
			if tc.expectedErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(tc.expectedErr, err)
			}

			if tc.expectedResp != nil {
				s.Require().Equal(tc.expectedResp, results.GetScanStats())
			}
		})
	}
}

func (s *ComplianceResultsStatsServiceTestSuite) TestGetComplianceOverallClusterStats() {
	testCases := []struct {
		desc         string
		query        *apiV2.RawQuery
		expectedResp []*apiV2.ComplianceClusterOverallStats
		expectedErr  error
		setMocks     func()
	}{
		{
			desc:        "Empty query",
			query:       &apiV2.RawQuery{Query: ""},
			expectedErr: nil,
			expectedResp: []*apiV2.ComplianceClusterOverallStats{
				convertUtils.GetComplianceClusterV2Count(s.T(), fixtureconsts.Cluster1, nil),
				convertUtils.GetComplianceClusterV2Count(s.T(), fixtureconsts.Cluster2, nil),
				convertUtils.GetComplianceClusterV2Count(s.T(), fixtureconsts.Cluster3, nil),
			},
			setMocks: func() {
				expectedQ := search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery()

				results := []*datastore.ResultStatusCountByCluster{
					convertUtils.GetComplianceStorageClusterCount(s.T(), fixtureconsts.Cluster1),
					convertUtils.GetComplianceStorageClusterCount(s.T(), fixtureconsts.Cluster2),
					convertUtils.GetComplianceStorageClusterCount(s.T(), fixtureconsts.Cluster3),
				}

				s.resultDatastore.EXPECT().CountByField(gomock.Any(), search.EmptyQuery(), search.ClusterID)
				s.resultDatastore.EXPECT().ComplianceClusterStats(gomock.Any(), expectedQ).Return(results, nil).Times(1)
				s.integrationDS.EXPECT().GetComplianceIntegrationByCluster(gomock.Any(), fixtureconsts.Cluster1).Return([]*storage.ComplianceIntegration{integration1}, nil).Times(1)
				s.integrationDS.EXPECT().GetComplianceIntegrationByCluster(gomock.Any(), fixtureconsts.Cluster2).Return([]*storage.ComplianceIntegration{integration2}, nil).Times(1)
				s.integrationDS.EXPECT().GetComplianceIntegrationByCluster(gomock.Any(), fixtureconsts.Cluster3).Return([]*storage.ComplianceIntegration{integration3}, nil).Times(1)
			},
		},
		{
			desc:        "Query with search field",
			query:       &apiV2.RawQuery{Query: "Cluster ID:" + fixtureconsts.Cluster1},
			expectedErr: nil,
			expectedResp: []*apiV2.ComplianceClusterOverallStats{
				convertUtils.GetComplianceClusterV2Count(s.T(), fixtureconsts.Cluster1, nil),
			},
			setMocks: func() {
				expectedQ := search.NewQueryBuilder().AddStrings(search.ClusterID, fixtureconsts.Cluster1).
					WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery()

				countQuery := search.NewQueryBuilder().AddStrings(search.ClusterID, fixtureconsts.Cluster1).ProtoQuery()

				results := []*datastore.ResultStatusCountByCluster{
					convertUtils.GetComplianceStorageClusterCount(s.T(), fixtureconsts.Cluster1),
				}
				s.resultDatastore.EXPECT().CountByField(gomock.Any(), countQuery, search.ClusterID)
				s.resultDatastore.EXPECT().ComplianceClusterStats(gomock.Any(), expectedQ).Return(results, nil).Times(1)
				s.integrationDS.EXPECT().GetComplianceIntegrationByCluster(gomock.Any(), fixtureconsts.Cluster1).Return([]*storage.ComplianceIntegration{integration1}, nil).Times(1)
			},
		},
		{
			desc:        "Query with non-existent field",
			query:       &apiV2.RawQuery{Query: "Cluster ID:id"},
			expectedErr: errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance cluster scan stats for query %v", &apiV2.RawQuery{Query: "Cluster ID:id"}),
			setMocks: func() {
				expectedQ := search.NewQueryBuilder().AddStrings(search.ClusterID, "id").
					WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery()
				countQuery := search.NewQueryBuilder().AddStrings(search.ClusterID, "id").ProtoQuery()

				s.resultDatastore.EXPECT().CountByField(gomock.Any(), countQuery, search.ClusterID)
				s.resultDatastore.EXPECT().ComplianceClusterStats(gomock.Any(), expectedQ).Return(nil, nil).Times(1)
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			tc.setMocks()

			results, err := s.service.GetComplianceOverallClusterStats(s.ctx, tc.query)
			if tc.expectedErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(tc.expectedErr, err)
			}

			if tc.expectedResp != nil {
				s.Require().Equal(tc.expectedResp, results.GetScanStats())
			}
		})
	}
}

func (s *ComplianceResultsStatsServiceTestSuite) TestGetComplianceClusterStats() {
	testCases := []struct {
		desc         string
		request      *apiV2.ComplianceProfileResultsRequest
		expectedResp []*apiV2.ComplianceClusterOverallStats
		expectedErr  error
		setMocks     func()
	}{
		{
			desc: "Empty query just profile",
			request: &apiV2.ComplianceProfileResultsRequest{
				ProfileName: "test-profile",
			},
			expectedErr: nil,
			expectedResp: []*apiV2.ComplianceClusterOverallStats{
				convertUtils.GetComplianceClusterV2Count(s.T(), fixtureconsts.Cluster1, scan1.LastExecutedTime),
				convertUtils.GetComplianceClusterV2Count(s.T(), fixtureconsts.Cluster2, scan2.LastExecutedTime),
				convertUtils.GetComplianceClusterV2Count(s.T(), fixtureconsts.Cluster3, scan3.LastExecutedTime),
			},
			setMocks: func() {
				expectedQ := search.ConjunctionQuery(
					search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, "test-profile").ProtoQuery(),
					search.EmptyQuery(),
				)

				countQuery := expectedQ.Clone()

				expectedQ.Pagination = &v1.QueryPagination{Limit: maxPaginationLimit}

				results := []*datastore.ResultStatusCountByCluster{
					convertUtils.GetComplianceStorageClusterCount(s.T(), fixtureconsts.Cluster1),
					convertUtils.GetComplianceStorageClusterCount(s.T(), fixtureconsts.Cluster2),
					convertUtils.GetComplianceStorageClusterCount(s.T(), fixtureconsts.Cluster3),
				}
				s.resultDatastore.EXPECT().CountByField(gomock.Any(), countQuery, search.ClusterID)
				s.resultDatastore.EXPECT().ComplianceClusterStats(gomock.Any(), expectedQ).Return(results, nil).Times(1)
				s.integrationDS.EXPECT().GetComplianceIntegrationByCluster(gomock.Any(), fixtureconsts.Cluster1).Return([]*storage.ComplianceIntegration{integration1}, nil).Times(1)
				s.integrationDS.EXPECT().GetComplianceIntegrationByCluster(gomock.Any(), fixtureconsts.Cluster2).Return([]*storage.ComplianceIntegration{integration2}, nil).Times(1)
				s.integrationDS.EXPECT().GetComplianceIntegrationByCluster(gomock.Any(), fixtureconsts.Cluster3).Return([]*storage.ComplianceIntegration{integration3}, nil).Times(1)

				scanQuery1 := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, "test-profile").AddExactMatches(search.ClusterID, fixtureconsts.Cluster1).ProtoQuery()
				scanQuery2 := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, "test-profile").AddExactMatches(search.ClusterID, fixtureconsts.Cluster2).ProtoQuery()
				scanQuery3 := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, "test-profile").AddExactMatches(search.ClusterID, fixtureconsts.Cluster3).ProtoQuery()
				s.scanDS.EXPECT().SearchScans(gomock.Any(), scanQuery1).Return([]*storage.ComplianceOperatorScanV2{scan1}, nil).Times(1)
				s.scanDS.EXPECT().SearchScans(gomock.Any(), scanQuery2).Return([]*storage.ComplianceOperatorScanV2{scan2}, nil).Times(1)
				s.scanDS.EXPECT().SearchScans(gomock.Any(), scanQuery3).Return([]*storage.ComplianceOperatorScanV2{scan3}, nil).Times(1)
			},
		},
		{
			desc: "Query with search field",
			request: &apiV2.ComplianceProfileResultsRequest{
				ProfileName: "test-profile",
				Query:       &apiV2.RawQuery{Query: "Cluster ID:" + fixtureconsts.Cluster1},
			},
			expectedErr: nil,
			expectedResp: []*apiV2.ComplianceClusterOverallStats{
				convertUtils.GetComplianceClusterV2Count(s.T(), fixtureconsts.Cluster1, scan1.LastExecutedTime),
			},
			setMocks: func() {
				expectedQ := search.ConjunctionQuery(
					search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, "test-profile").ProtoQuery(),
					search.NewQueryBuilder().AddStrings(search.ClusterID, fixtureconsts.Cluster1).ProtoQuery(),
				)

				countQuery := expectedQ.Clone()

				expectedQ.Pagination = &v1.QueryPagination{Limit: maxPaginationLimit}

				results := []*datastore.ResultStatusCountByCluster{
					convertUtils.GetComplianceStorageClusterCount(s.T(), fixtureconsts.Cluster1),
				}
				s.resultDatastore.EXPECT().CountByField(gomock.Any(), countQuery, search.ClusterID)
				s.resultDatastore.EXPECT().ComplianceClusterStats(gomock.Any(), expectedQ).Return(results, nil).Times(1)
				s.integrationDS.EXPECT().GetComplianceIntegrationByCluster(gomock.Any(), fixtureconsts.Cluster1).Return([]*storage.ComplianceIntegration{integration1}, nil).Times(1)

				scanQuery1 := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, "test-profile").AddExactMatches(search.ClusterID, fixtureconsts.Cluster1).ProtoQuery()
				s.scanDS.EXPECT().SearchScans(gomock.Any(), scanQuery1).Return([]*storage.ComplianceOperatorScanV2{scan1}, nil).Times(1)
			},
		},
		{
			desc: "Query with non-existent field",
			request: &apiV2.ComplianceProfileResultsRequest{
				Query: &apiV2.RawQuery{Query: "Cluster ID:id"},
			},
			expectedErr: errors.Wrap(errox.InvalidArgs, "Profile name is required"),
			setMocks: func() {

			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			tc.setMocks()

			results, err := s.service.GetComplianceClusterStats(s.ctx, tc.request)
			if tc.expectedErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(tc.expectedErr, err)
			}

			if tc.expectedResp != nil {
				s.Require().Equal(tc.expectedResp, results.GetScanStats())
			}
		})
	}
}

func (s *ComplianceResultsStatsServiceTestSuite) TestGetComplianceProfileScanStats() {
	testCases := []struct {
		desc         string
		query        *apiV2.RawQuery
		expectedResp []*apiV2.ComplianceProfileScanStats
		expectedErr  error
		setMocks     func()
	}{
		{
			desc:        "Empty query",
			query:       &apiV2.RawQuery{Query: ""},
			expectedErr: nil,
			expectedResp: []*apiV2.ComplianceProfileScanStats{
				convertUtils.GetComplianceProfileScanV2Count(s.T(), "ocp4"),
				convertUtils.GetComplianceProfileScanV2Count(s.T(), "rhcos4-moderate"),
				convertUtils.GetComplianceProfileScanV2Count(s.T(), "ocp4-node"),
			},
			setMocks: func() {
				expectedQ := search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery()
				results := []*datastore.ResourceResultCountByProfile{
					convertUtils.GetComplianceStorageProfileScanCount(s.T(), "ocp4"),
					convertUtils.GetComplianceStorageProfileScanCount(s.T(), "rhcos4-moderate"),
					convertUtils.GetComplianceStorageProfileScanCount(s.T(), "ocp4-node"),
				}
				s.resultDatastore.EXPECT().CountByField(gomock.Any(), search.EmptyQuery(), search.ComplianceOperatorProfileName)
				s.resultDatastore.EXPECT().ComplianceProfileResultStats(gomock.Any(), expectedQ).Return(results, nil).Times(1)
				profilesOcp := []*storage.ComplianceOperatorProfileV2{{
					Name:           "ocp4",
					ProfileVersion: "test_version_ocp4",
					Title:          "test_title_ocp4",
				}}
				s.profileDS.EXPECT().SearchProfiles(gomock.Any(), search.NewQueryBuilder().
					AddExactMatches(search.ComplianceOperatorProfileName, "ocp4").ProtoQuery()).Return(profilesOcp, nil).Times(1)
				profiles := []*storage.ComplianceOperatorProfileV2{{
					Name:           "rhcos4-moderate",
					ProfileVersion: "test_version_rhcos4-moderate",
					Title:          "test_title_rhcos4-moderate",
				}}
				s.profileDS.EXPECT().SearchProfiles(gomock.Any(), search.NewQueryBuilder().
					AddExactMatches(search.ComplianceOperatorProfileName, "rhcos4-moderate").ProtoQuery()).Return(profiles, nil).Times(1)
				profiles = []*storage.ComplianceOperatorProfileV2{{
					Name:           "ocp4-node",
					ProfileVersion: "test_version_ocp4-node",
					Title:          "test_title_ocp4-node",
				}}
				s.profileDS.EXPECT().SearchProfiles(gomock.Any(), search.NewQueryBuilder().
					AddExactMatches(search.ComplianceOperatorProfileName, "ocp4-node").ProtoQuery()).Return(profiles, nil).Times(1)
			},
		},
		{
			desc:        "Query with search field",
			query:       &apiV2.RawQuery{Query: "Cluster ID:" + fixtureconsts.Cluster1},
			expectedErr: nil,
			expectedResp: []*apiV2.ComplianceProfileScanStats{
				convertUtils.GetComplianceProfileScanV2Count(s.T(), "ocp4-node"),
			},
			setMocks: func() {
				expectedQ := search.NewQueryBuilder().AddStrings(search.ClusterID, fixtureconsts.Cluster1).
					WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery()
				countQuery := search.NewQueryBuilder().AddStrings(search.ClusterID, fixtureconsts.Cluster1).ProtoQuery()

				results := []*datastore.ResourceResultCountByProfile{
					convertUtils.GetComplianceStorageProfileScanCount(s.T(), "ocp4-node"),
				}
				s.resultDatastore.EXPECT().CountByField(gomock.Any(), countQuery, search.ComplianceOperatorProfileName)
				s.resultDatastore.EXPECT().ComplianceProfileResultStats(gomock.Any(), expectedQ).Return(results, nil).Times(1)
				profiles := []*storage.ComplianceOperatorProfileV2{{
					Name:           "ocp4-node",
					ProfileVersion: "test_version_ocp4-node",
					Title:          "test_title_ocp4-node",
				}}
				s.profileDS.EXPECT().SearchProfiles(gomock.Any(), gomock.Any()).Return(profiles, nil).AnyTimes()
			},
		},
		{
			desc:        "Query with non-existent field",
			query:       &apiV2.RawQuery{Query: "Cluster ID:id"},
			expectedErr: errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance cluster scan stats for query %v", &apiV2.RawQuery{Query: "Cluster ID:id"}),
			setMocks: func() {
				expectedQ := search.NewQueryBuilder().AddStrings(search.ClusterID, "id").
					WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery()
				countQuery := search.NewQueryBuilder().AddStrings(search.ClusterID, "id").ProtoQuery()

				s.resultDatastore.EXPECT().CountByField(gomock.Any(), countQuery, search.ComplianceOperatorProfileName)
				s.resultDatastore.EXPECT().ComplianceProfileResultStats(gomock.Any(), expectedQ).Return(nil, nil).Times(1)
				s.profileDS.EXPECT().SearchProfiles(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			tc.setMocks()

			results, err := s.service.GetComplianceProfilesStats(s.ctx, tc.query)
			if tc.expectedErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(tc.expectedErr, err)
			}

			if tc.expectedResp != nil {
				s.Require().ElementsMatch(tc.expectedResp, results.GetScanStats())
			}
		})
	}
}

func (s *ComplianceResultsStatsServiceTestSuite) TestGetComplianceProfileStats() {
	testCases := []struct {
		desc         string
		query        *apiV2.ComplianceProfileResultsRequest
		expectedResp []*apiV2.ComplianceProfileScanStats
		expectedErr  error
		setMocks     func()
	}{
		{
			desc: "Empty query",
			query: &apiV2.ComplianceProfileResultsRequest{
				ProfileName: "ocp4",
				Query:       &apiV2.RawQuery{Query: ""},
			},
			expectedErr: nil,
			expectedResp: []*apiV2.ComplianceProfileScanStats{
				convertUtils.GetComplianceProfileScanV2Count(s.T(), "ocp4"),
			},
			setMocks: func() {

				expectedQ := search.ConjunctionQuery(
					search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, "ocp4").ProtoQuery(),
					search.EmptyQuery(),
				)
				countQuery := expectedQ.Clone()
				expectedQ.Pagination = &v1.QueryPagination{Limit: maxPaginationLimit}

				results := []*datastore.ResourceResultCountByProfile{
					convertUtils.GetComplianceStorageProfileScanCount(s.T(), "ocp4"),
				}
				s.resultDatastore.EXPECT().CountByField(gomock.Any(), countQuery, search.ComplianceOperatorProfileName)
				s.resultDatastore.EXPECT().ComplianceProfileResultStats(gomock.Any(), expectedQ).Return(results, nil).Times(1)
				profilesOcp := []*storage.ComplianceOperatorProfileV2{{
					Name:           "ocp4",
					ProfileVersion: "test_version_ocp4",
					Title:          "test_title_ocp4",
				}}
				s.profileDS.EXPECT().SearchProfiles(gomock.Any(), search.NewQueryBuilder().
					AddExactMatches(search.ComplianceOperatorProfileName, "ocp4").ProtoQuery()).Return(profilesOcp, nil).Times(1)
			},
		},
		{
			desc: "Query with search field",
			query: &apiV2.ComplianceProfileResultsRequest{
				ProfileName: "ocp4-node",
				Query:       &apiV2.RawQuery{Query: "Cluster ID:" + fixtureconsts.Cluster1},
			},
			expectedErr: nil,
			expectedResp: []*apiV2.ComplianceProfileScanStats{
				convertUtils.GetComplianceProfileScanV2Count(s.T(), "ocp4-node"),
			},
			setMocks: func() {
				expectedQ := search.NewQueryBuilder().AddStrings(search.ClusterID, fixtureconsts.Cluster1).ProtoQuery()
				expectedQ = search.ConjunctionQuery(
					search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, "ocp4-node").ProtoQuery(),
					expectedQ,
				)
				countQuery := expectedQ.Clone()
				expectedQ.Pagination = &v1.QueryPagination{Limit: maxPaginationLimit}

				results := []*datastore.ResourceResultCountByProfile{
					convertUtils.GetComplianceStorageProfileScanCount(s.T(), "ocp4-node"),
				}
				s.resultDatastore.EXPECT().CountByField(gomock.Any(), countQuery, search.ComplianceOperatorProfileName)
				s.resultDatastore.EXPECT().ComplianceProfileResultStats(gomock.Any(), expectedQ).Return(results, nil).Times(1)
				profiles := []*storage.ComplianceOperatorProfileV2{{
					Name:           "ocp4-node",
					ProfileVersion: "test_version_ocp4-node",
					Title:          "test_title_ocp4-node",
				}}
				s.profileDS.EXPECT().SearchProfiles(gomock.Any(), search.NewQueryBuilder().
					AddExactMatches(search.ComplianceOperatorProfileName, "ocp4-node").ProtoQuery()).Return(profiles, nil).Times(1)
			},
		},
		{
			desc: "Query with non-existent field",
			query: &apiV2.ComplianceProfileResultsRequest{
				ProfileName: "",
				Query:       &apiV2.RawQuery{Query: "Cluster ID:" + fixtureconsts.Cluster1},
			},
			expectedErr: errors.Wrap(errox.InvalidArgs, "Profile name is required"),
			setMocks: func() {
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			tc.setMocks()

			results, err := s.service.GetComplianceProfileStats(s.ctx, tc.query)
			if tc.expectedErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(tc.expectedErr, err)
			}

			if tc.expectedResp != nil {
				s.Require().Equal(tc.expectedResp, results.GetScanStats())
			}
		})
	}
}

func (s *ComplianceResultsStatsServiceTestSuite) TestGetComplianceProfileCheckStats() {
	testCases := []struct {
		desc         string
		query        *apiV2.ComplianceProfileCheckRequest
		expectedResp *apiV2.ListComplianceProfileResults
		expectedErr  error
		setMocks     func()
	}{
		{
			desc: "Empty query",
			query: &apiV2.ComplianceProfileCheckRequest{
				ProfileName: "ocp4",
				CheckName:   "check-name",
				Query:       &apiV2.RawQuery{Query: ""},
			},
			expectedErr:  nil,
			expectedResp: convertUtils.GetComplianceProfileResultsV2(s.T(), "ocp4"),
			setMocks: func() {
				expectedQ := search.ConjunctionQuery(
					search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, "ocp4").
						AddExactMatches(search.ComplianceOperatorCheckName, "check-name").ProtoQuery(),
					search.EmptyQuery(),
				)
				expectedQ.Pagination = &v1.QueryPagination{Limit: maxPaginationLimit}

				results := []*datastore.ResourceResultsByProfile{
					convertUtils.GetComplianceStorageProfileResults(s.T(), "ocp4"),
				}
				s.resultDatastore.EXPECT().ComplianceProfileResults(gomock.Any(), expectedQ).Return(results, nil).Times(1)
			},
		},
		{
			desc: "Query with search field",
			query: &apiV2.ComplianceProfileCheckRequest{
				ProfileName: "ocp4",
				CheckName:   "check-name",
				Query:       &apiV2.RawQuery{Query: "Cluster ID:" + fixtureconsts.Cluster1},
			},
			expectedErr:  nil,
			expectedResp: convertUtils.GetComplianceProfileResultsV2(s.T(), "ocp4"),
			setMocks: func() {
				expectedQ := search.NewQueryBuilder().AddStrings(search.ClusterID, fixtureconsts.Cluster1).ProtoQuery()
				expectedQ = search.ConjunctionQuery(
					search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, "ocp4").
						AddExactMatches(search.ComplianceOperatorCheckName, "check-name").ProtoQuery(),
					expectedQ,
				)
				expectedQ.Pagination = &v1.QueryPagination{Limit: maxPaginationLimit}

				results := []*datastore.ResourceResultsByProfile{
					convertUtils.GetComplianceStorageProfileResults(s.T(), "ocp4"),
				}
				s.resultDatastore.EXPECT().ComplianceProfileResults(gomock.Any(), expectedQ).Return(results, nil).Times(1)
			},
		},
		{
			desc: "Query with non-existent field",
			query: &apiV2.ComplianceProfileCheckRequest{
				ProfileName: "",
				Query:       &apiV2.RawQuery{Query: "Cluster ID:" + fixtureconsts.Cluster1},
			},
			expectedErr: errors.Wrap(errox.InvalidArgs, "Profile name is required"),
			setMocks: func() {
			},
		},
		{
			desc: "Query with non-existent field",
			query: &apiV2.ComplianceProfileCheckRequest{
				ProfileName: "ocp4",
				Query:       &apiV2.RawQuery{Query: "Cluster ID:" + fixtureconsts.Cluster1},
			},
			expectedErr: errors.Wrap(errox.InvalidArgs, "Compliance check name is required"),
			setMocks: func() {
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			tc.setMocks()

			results, err := s.service.GetComplianceProfileCheckStats(s.ctx, tc.query)
			if tc.expectedErr == nil {
				s.Require().NoError(err)
				s.Require().Equal(tc.expectedResp, results)
			} else {
				s.Require().Error(tc.expectedErr, err)
				s.Require().Nil(results)
			}
		})
	}
}

func getTestRec(scanName string) *storage.ComplianceOperatorScanConfigurationV2 {
	return &storage.ComplianceOperatorScanConfigurationV2{
		Id:                     scanName,
		ScanConfigName:         scanName,
		AutoApplyRemediations:  false,
		AutoUpdateRemediations: false,
		OneTimeScan:            false,
		Profiles: []*storage.ComplianceOperatorScanConfigurationV2_ProfileName{
			{
				ProfileName: "ocp4-cis",
			},
		},
		StrictNodeScan: false,
		Description:    "test-description",
		Clusters: []*storage.ComplianceOperatorScanConfigurationV2_Cluster{
			{
				ClusterId: fixtureconsts.Cluster1,
			},
			{
				ClusterId: fixtureconsts.Cluster2,
			},
		},
	}
}
