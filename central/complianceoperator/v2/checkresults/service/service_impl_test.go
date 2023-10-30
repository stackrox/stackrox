package service

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	resultMocks "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore/mocks"
	convertUtils "github.com/stackrox/rox/central/convert/testutils"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

func TestComplianceScanConfigService(t *testing.T) {
	suite.Run(t, new(ComplianceResultsServiceTestSuite))
}

type ComplianceResultsServiceTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	ctx             context.Context
	resultDatastore *resultMocks.MockDataStore
	service         Service
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

	s.service = New(s.resultDatastore)
}

func (s *ComplianceResultsServiceTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ComplianceResultsServiceTestSuite) TestGetComplianceScanResults() {
	testCases := []struct {
		desc         string
		query        *apiV2.RawQuery
		expectedResp []*apiV2.ComplianceScanResult
		expectedErr  error
		found        bool
		setMocks     func()
	}{
		{
			desc:         "Empty query",
			query:        &apiV2.RawQuery{Query: ""},
			expectedErr:  nil,
			expectedResp: convertUtils.GetConvertedComplianceResults(s.T()),
			found:        true,
			setMocks: func() {
				expectedQ := search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery()
				s.resultDatastore.EXPECT().SearchComplianceCheckResults(gomock.Any(), expectedQ).Return(convertUtils.GetComplianceStorageResults(s.T()), nil).Times(1)
			},
		},
		{
			desc:        "Query with search field",
			query:       &apiV2.RawQuery{Query: "Cluster ID:" + fixtureconsts.Cluster1},
			expectedErr: nil,
			found:       true,
			setMocks: func() {
				expectedQ := search.NewQueryBuilder().AddStrings(search.ClusterID, fixtureconsts.Cluster1).
					WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery()

				s.resultDatastore.EXPECT().SearchComplianceCheckResults(gomock.Any(), expectedQ).Return(convertUtils.GetOneClusterComplianceStorageResults(s.T(), fixtureconsts.Cluster1), nil).Times(1)
			},
		},
		{
			desc: "Query with custom pagination",
			query: &apiV2.RawQuery{
				Query:      "",
				Pagination: &apiV2.Pagination{Limit: 1},
			},
			expectedErr: nil,
			found:       true,
			setMocks: func() {
				expectedQ := search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(1)).ProtoQuery()
				returnResults := []*storage.ComplianceOperatorCheckResultV2{
					convertUtils.GetComplianceStorageResults(s.T())[0],
				}

				s.resultDatastore.EXPECT().SearchComplianceCheckResults(gomock.Any(), expectedQ).Return(returnResults, nil).Times(1)
			},
		},
		{
			desc:        "Query with non-existent field",
			query:       &apiV2.RawQuery{Query: "Cluster ID:id"},
			expectedErr: errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance scan results for query %v", &apiV2.RawQuery{Query: "Cluster ID:id"}),
			found:       false,
			setMocks: func() {
				expectedQ := search.NewQueryBuilder().AddStrings(search.ClusterID, "id").
					WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery()

				s.resultDatastore.EXPECT().SearchComplianceCheckResults(gomock.Any(), expectedQ).Return(nil, nil).Times(1)
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
				s.Require().Equal(convertUtils.GetConvertedComplianceResults(s.T()), results.GetScanResults())
			}
		})
	}
}

func (s *ComplianceResultsServiceTestSuite) TestGetComplianceClusterScanStats() {
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
				convertUtils.GetComplianceClusterV2Count(s.T(), fixtureconsts.Cluster1),
				convertUtils.GetComplianceClusterV2Count(s.T(), fixtureconsts.Cluster2),
				convertUtils.GetComplianceClusterV2Count(s.T(), fixtureconsts.Cluster3),
			},
			setMocks: func() {
				expectedQ := search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery()
				results := []*datastore.ResourceCountByResultByCluster{
					convertUtils.GetComplianceStorageCount(s.T(), fixtureconsts.Cluster1),
					convertUtils.GetComplianceStorageCount(s.T(), fixtureconsts.Cluster2),
					convertUtils.GetComplianceStorageCount(s.T(), fixtureconsts.Cluster3),
				}
				s.resultDatastore.EXPECT().ComplianceCheckResultStats(gomock.Any(), expectedQ).Return(results, nil).Times(1)
			},
		},
		{
			desc:        "Query with search field",
			query:       &apiV2.RawQuery{Query: "Cluster ID:" + fixtureconsts.Cluster1},
			expectedErr: nil,
			expectedResp: []*apiV2.ComplianceClusterScanStats{
				convertUtils.GetComplianceClusterV2Count(s.T(), fixtureconsts.Cluster1),
			},
			setMocks: func() {
				expectedQ := search.NewQueryBuilder().AddStrings(search.ClusterID, fixtureconsts.Cluster1).
					WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery()

				results := []*datastore.ResourceCountByResultByCluster{
					convertUtils.GetComplianceStorageCount(s.T(), fixtureconsts.Cluster1),
				}
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
