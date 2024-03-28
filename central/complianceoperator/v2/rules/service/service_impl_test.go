package service

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	profileMocks "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore/mocks"
	convertUtils "github.com/stackrox/rox/central/convert/testutils"
	apiV1 "github.com/stackrox/rox/generated/api/v1"
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

func TestComplianceProfileService(t *testing.T) {
	suite.Run(t, new(ComplianceProfilesServiceTestSuite))
}

type ComplianceProfilesServiceTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	ctx              context.Context
	profileDatastore *profileMocks.MockDataStore
	service          Service
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

	s.service = New(s.profileDatastore)
}

func (s *ComplianceProfilesServiceTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ComplianceProfilesServiceTestSuite) TestGetComplianceProfile() {
	profileID := "ocp-cis-4.2"
	s.profileDatastore.EXPECT().GetProfile(s.ctx, profileID).Return(convertUtils.GetProfileV2Storage(s.T()), true, nil)

	profile, err := s.service.GetComplianceProfile(s.ctx, &apiV2.ResourceByID{Id: profileID})
	s.Require().NoError(err)
	s.Require().Equal(convertUtils.GetProfileV2Api(s.T()), profile)
}

func (s *ComplianceProfilesServiceTestSuite) TestGetComplianceProfileNotFound() {
	// Profile does not exist
	profileID := "ocp-cis-4.2"
	s.profileDatastore.EXPECT().GetProfile(s.ctx, profileID).Return(nil, false, nil)

	profile, err := s.service.GetComplianceProfile(s.ctx, &apiV2.ResourceByID{Id: profileID})
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
			query:        &apiV2.ProfilesForClusterRequest{ClusterId: fixtureconsts.Cluster1},
			expectedErr:  nil,
			expectedResp: convertUtils.GetProfilesV2Api(s.T()),
			found:        true,
			setMocks: func() {
				s.profileDatastore.EXPECT().GetProfilesByClusters(gomock.Any(), []string{fixtureconsts.Cluster1}).Return(convertUtils.GetProfilesV2Storage(s.T()), nil).Times(1)
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
				s.Require().Equal(tc.expectedResp, results.GetProfiles())
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
			query:       &apiV2.ClustersProfileSummaryRequest{ClusterIds: []string{fixtureconsts.Cluster1}},
			expectedErr: nil,
			expectedResp: []*apiV2.ComplianceProfileSummary{
				{
					Name:           "ocp4",
					ProductType:    "platform",
					Description:    "this is a test",
					Title:          "A Title",
					ProfileVersion: "version 1",
					RuleCount:      5,
				},
			},
			found: true,
			setMocks: func() {
				s.profileDatastore.EXPECT().GetProfilesByClusters(gomock.Any(), []string{fixtureconsts.Cluster1}).Return([]*storage.ComplianceOperatorProfileV2{
					{
						Name:           "ocp4",
						ProductType:    "platform",
						Description:    "this is a test",
						Title:          "A Title",
						ProfileVersion: "version 1",
						Rules: []*storage.ComplianceOperatorProfileV2_Rule{
							{
								RuleName: "test 1",
							},
							{
								RuleName: "test 2",
							},
							{
								RuleName: "test 3",
							},
							{
								RuleName: "test 4",
							},
							{
								RuleName: "test 5",
							},
						},
					},
				}, nil).Times(1)
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
				s.Require().Equal(tc.expectedResp, results.GetProfiles())
			}
		})
	}
}

func (s *ComplianceProfilesServiceTestSuite) TestCountComplianceProfiles() {
	allAccessContext := sac.WithAllAccess(context.Background())

	testCases := []struct {
		desc      string
		query     *apiV2.RawQuery
		expectedQ *apiV1.Query
	}{
		{
			desc:      "Empty query",
			query:     &apiV2.RawQuery{Query: ""},
			expectedQ: search.NewQueryBuilder().ProtoQuery(),
		},
		{
			desc:      "Query with search field",
			query:     &apiV2.RawQuery{Query: "Compliance Profile Name:ocp-4"},
			expectedQ: search.NewQueryBuilder().AddStrings(search.ComplianceOperatorProfileName, "ocp-4").ProtoQuery(),
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {

			s.profileDatastore.EXPECT().CountProfiles(allAccessContext, tc.expectedQ).
				Return(1, nil).Times(1)

			profiles, err := s.service.GetComplianceProfileCount(allAccessContext, tc.query)
			s.Require().NoError(err)
			s.Require().Equal(int32(1), profiles.Count)
		})
	}

}
