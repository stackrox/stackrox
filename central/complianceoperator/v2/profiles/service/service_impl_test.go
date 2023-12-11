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
		query        *apiV2.RawQuery
		expectedResp []*apiV2.ComplianceProfile
		expectedErr  error
		found        bool
		setMocks     func()
	}{
		{
			desc:         "Empty query",
			query:        &apiV2.RawQuery{Query: ""},
			expectedErr:  nil,
			expectedResp: convertUtils.GetProfilesV2Api(s.T()),
			found:        true,
			setMocks: func() {
				expectedQ := search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery()
				s.profileDatastore.EXPECT().SearchProfiles(gomock.Any(), expectedQ).Return(convertUtils.GetProfilesV2Storage(s.T()), nil).Times(1)
			},
		},
		{
			desc:         "Query with search field",
			query:        &apiV2.RawQuery{Query: "Cluster ID:" + fixtureconsts.Cluster1},
			expectedErr:  nil,
			expectedResp: convertUtils.GetProfilesV2Api(s.T()),
			found:        true,
			setMocks: func() {
				expectedQ := search.NewQueryBuilder().AddStrings(search.ClusterID, fixtureconsts.Cluster1).
					WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery()

				s.profileDatastore.EXPECT().SearchProfiles(gomock.Any(), expectedQ).Return(convertUtils.GetProfilesV2Storage(s.T()), nil).Times(1)
			},
		},
		{
			desc: "Query with custom pagination",
			query: &apiV2.RawQuery{
				Query:      "",
				Pagination: &apiV2.Pagination{Limit: 1},
			},
			expectedErr: nil,
			expectedResp: []*apiV2.ComplianceProfile{
				convertUtils.GetProfileV2Api(s.T()),
			},
			found: true,
			setMocks: func() {
				expectedQ := search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(1)).ProtoQuery()
				returnResults := []*storage.ComplianceOperatorProfileV2{
					convertUtils.GetProfileV2Storage(s.T()),
				}

				s.profileDatastore.EXPECT().SearchProfiles(gomock.Any(), expectedQ).Return(returnResults, nil).Times(1)
			},
		},
		{
			desc:         "Query with non-existent field",
			query:        &apiV2.RawQuery{Query: "Cluster ID:id"},
			expectedErr:  errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance scan results for query %v", &apiV2.RawQuery{Query: "Cluster ID:id"}),
			expectedResp: nil,
			found:        false,
			setMocks: func() {
				expectedQ := search.NewQueryBuilder().AddStrings(search.ClusterID, "id").
					WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery()

				s.profileDatastore.EXPECT().SearchProfiles(gomock.Any(), expectedQ).Return(nil, nil).Times(1)
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
