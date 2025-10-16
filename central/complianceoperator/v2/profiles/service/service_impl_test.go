package service

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	benchmarkMocks "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/datastore/mocks"
	profileMocks "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore/mocks"
	convertUtils "github.com/stackrox/rox/central/convert/testutils"
	apiV1 "github.com/stackrox/rox/generated/api/v1"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

func TestComplianceProfileService(t *testing.T) {
	suite.Run(t, new(ComplianceProfilesServiceTestSuite))
}

type ComplianceProfilesServiceTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	ctx                context.Context
	profileDatastore   *profileMocks.MockDataStore
	benchmarkDatastore *benchmarkMocks.MockDataStore
	service            Service
}

func (s *ComplianceProfilesServiceTestSuite) SetupSuite() {
	s.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")
	if !features.ComplianceEnhancements.Enabled() {
		s.T().Skip("Skip test when compliance enhancements are disabled")
		s.T().SkipNow()
	}

	s.ctx = sac.WithAllAccess(context.Background())
}

func (s *ComplianceProfilesServiceTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.profileDatastore = profileMocks.NewMockDataStore(s.mockCtrl)
	s.benchmarkDatastore = benchmarkMocks.NewMockDataStore(s.mockCtrl)

	s.service = New(s.profileDatastore, s.benchmarkDatastore)
}

func (s *ComplianceProfilesServiceTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ComplianceProfilesServiceTestSuite) TestGetComplianceProfile() {
	profileID := "ocp-cis-4.2"
	testProfile := convertUtils.GetProfileV2Storage(s.T())
	s.profileDatastore.EXPECT().GetProfile(s.ctx, profileID).Return(testProfile, true, nil)

	cobv2 := &storage.ComplianceOperatorBenchmarkV2{}
	cobv2.SetId(uuid.NewV4().String())
	cobv2.SetName("CIS")
	cobv2.SetShortName("OCP_CIS")
	cobv2.SetVersion("1-5")
	s.benchmarkDatastore.EXPECT().GetBenchmarksByProfileName(s.ctx, testProfile.GetName()).Return([]*storage.ComplianceOperatorBenchmarkV2{cobv2}, nil).Times(1)

	rbid := &apiV2.ResourceByID{}
	rbid.SetId(profileID)
	profile, err := s.service.GetComplianceProfile(s.ctx, rbid)
	s.Require().NoError(err)
	protoassert.Equal(s.T(), convertUtils.GetProfileV2Api(s.T()), profile)
}

func (s *ComplianceProfilesServiceTestSuite) TestGetComplianceProfileNotFound() {
	// Profile does not exist
	profileID := "ocp-cis-4.2"
	s.profileDatastore.EXPECT().GetProfile(s.ctx, profileID).Return(nil, false, nil)

	rbid := &apiV2.ResourceByID{}
	rbid.SetId(profileID)
	profile, err := s.service.GetComplianceProfile(s.ctx, rbid)
	s.Require().Error(err)
	s.Require().Empty(profile)

}

func (s *ComplianceProfilesServiceTestSuite) TestListComplianceProfiles() {
	testCases := []struct {
		desc         string
		query        *apiV2.ProfilesForClusterRequest
		expectedResp []*apiV2.ComplianceProfile
		expectedErr  error
		found        bool
		setMocks     func()
	}{
		{
			desc:         "Empty query",
			query:        &apiV2.ProfilesForClusterRequest{},
			expectedErr:  errors.Wrap(errox.InvalidArgs, "cluster is required"),
			expectedResp: []*apiV2.ComplianceProfile(nil),
			found:        true,
			setMocks: func() {
			},
		},
		{
			desc:         "Query with cluster 1",
			query:        apiV2.ProfilesForClusterRequest_builder{ClusterId: fixtureconsts.Cluster1}.Build(),
			expectedErr:  nil,
			expectedResp: convertUtils.GetProfilesV2Api(s.T()),
			found:        true,
			setMocks: func() {
				profileQuery := search.ConjunctionQuery(
					search.NewQueryBuilder().AddSelectFields().AddExactMatches(search.ClusterID, fixtureconsts.Cluster1).ProtoQuery(),
					search.EmptyQuery(),
				)
				countQuery := profileQuery.CloneVT()
				paginated.FillPaginationV2(profileQuery, nil, maxPaginationLimit)

				profiles := convertUtils.GetProfilesV2Storage(s.T())
				s.profileDatastore.EXPECT().SearchProfiles(gomock.Any(), profileQuery).Return(profiles, nil).Times(1)
				s.profileDatastore.EXPECT().CountProfiles(gomock.Any(), countQuery).Return(1, nil).Times(1)

				for _, profile := range profiles {
					s.benchmarkDatastore.EXPECT().GetBenchmarksByProfileName(s.ctx, profile.GetName()).Return([]*storage.ComplianceOperatorBenchmarkV2{storage.ComplianceOperatorBenchmarkV2_builder{
						Id:        uuid.NewV4().String(),
						Name:      "CIS",
						ShortName: "OCP_CIS",
						Version:   "1-5",
					}.Build()}, nil).Times(1)
				}
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			tc.setMocks()

			results, err := s.service.ListComplianceProfiles(s.ctx, tc.query)
			if tc.expectedErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(tc.expectedErr, err)
			}

			if tc.expectedResp != nil {
				protoassert.SlicesEqual(s.T(), tc.expectedResp, results.GetProfiles())
			}
		})
	}
}

func (s *ComplianceProfilesServiceTestSuite) TestListProfileSummaries() {
	testCases := []struct {
		desc         string
		query        *apiV2.ClustersProfileSummaryRequest
		expectedResp []*apiV2.ComplianceProfileSummary
		expectedErr  error
		found        bool
		setMocks     func()
	}{
		{
			desc:         "Empty query",
			query:        &apiV2.ClustersProfileSummaryRequest{},
			expectedErr:  errors.Wrap(errox.InvalidArgs, "cluster is required"),
			expectedResp: []*apiV2.ComplianceProfileSummary(nil),
			found:        true,
			setMocks: func() {
			},
		},
		{
			desc:        "Query with cluster 1",
			query:       apiV2.ClustersProfileSummaryRequest_builder{ClusterIds: []string{fixtureconsts.Cluster1}}.Build(),
			expectedErr: nil,
			expectedResp: []*apiV2.ComplianceProfileSummary{
				apiV2.ComplianceProfileSummary_builder{
					Name:           "ocp4",
					ProductType:    "platform",
					Description:    "this is a test",
					Title:          "A Title",
					ProfileVersion: "version 1",
					RuleCount:      5,
					Standards: []*apiV2.ComplianceBenchmark{apiV2.ComplianceBenchmark_builder{
						Name:      "CIS",
						ShortName: "OCP_CIS",
						Version:   "1-5",
					}.Build()},
				}.Build(),
			},
			found: true,
			setMocks: func() {
				profileQuery := search.EmptyQuery()
				paginated.FillPaginationV2(profileQuery, nil, maxPaginationLimit)
				profileQuery.GetPagination().SetSortOptions([]*apiV1.QuerySortOption{
					apiV1.QuerySortOption_builder{
						Field: search.ComplianceOperatorProfileName.String(),
					}.Build(),
				})

				s.profileDatastore.EXPECT().GetProfilesNames(gomock.Any(), profileQuery, []string{fixtureconsts.Cluster1}).Return([]string{"ocp4"}, nil).Times(1)
				s.profileDatastore.EXPECT().CountDistinctProfiles(gomock.Any(), search.EmptyQuery(), []string{fixtureconsts.Cluster1}).Return(1, nil).Times(1)

				searchQuery := search.NewQueryBuilder().AddSelectFields().AddExactMatches(search.ComplianceOperatorProfileName, "ocp4").ProtoQuery()
				searchQuery.SetPagination(&apiV1.QueryPagination{})
				searchQuery.GetPagination().SetSortOptions(profileQuery.GetPagination().GetSortOptions())

				profiles := []*storage.ComplianceOperatorProfileV2{
					storage.ComplianceOperatorProfileV2_builder{
						Name:           "ocp4",
						ProductType:    "platform",
						Description:    "this is a test",
						Title:          "A Title",
						ProfileVersion: "version 1",
						Rules: []*storage.ComplianceOperatorProfileV2_Rule{
							storage.ComplianceOperatorProfileV2_Rule_builder{
								RuleName: "test 1",
							}.Build(),
							storage.ComplianceOperatorProfileV2_Rule_builder{
								RuleName: "test 2",
							}.Build(),
							storage.ComplianceOperatorProfileV2_Rule_builder{
								RuleName: "test 3",
							}.Build(),
							storage.ComplianceOperatorProfileV2_Rule_builder{
								RuleName: "test 4",
							}.Build(),
							storage.ComplianceOperatorProfileV2_Rule_builder{
								RuleName: "test 5",
							}.Build(),
						},
					}.Build(),
				}
				s.profileDatastore.EXPECT().SearchProfiles(gomock.Any(), searchQuery).Return(profiles, nil).Times(1)

				for _, profile := range profiles {
					s.benchmarkDatastore.EXPECT().GetBenchmarksByProfileName(s.ctx, profile.GetName()).Return([]*storage.ComplianceOperatorBenchmarkV2{storage.ComplianceOperatorBenchmarkV2_builder{
						Id:        uuid.NewV4().String(),
						Name:      "CIS",
						ShortName: "OCP_CIS",
						Version:   "1-5",
					}.Build()}, nil).Times(1)
				}
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			tc.setMocks()

			results, err := s.service.ListProfileSummaries(s.ctx, tc.query)
			if tc.expectedErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(tc.expectedErr, err)
			}

			if tc.expectedResp != nil {
				protoassert.SlicesEqual(s.T(), tc.expectedResp, results.GetProfiles())
			}
		})
	}
}
