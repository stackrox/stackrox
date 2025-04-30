package resolvers

import (
	"context"
	"math"
	"testing"

	"github.com/stackrox/rox/central/graphql/resolvers/inputtypes"
	"github.com/stackrox/rox/central/views/platformcve"
	platformCVEViewMock "github.com/stackrox/rox/central/views/platformcve/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestPlatformCVECoreResolver(t *testing.T) {
	suite.Run(t, new(PlatformCVEResolverTestSuite))
}

type PlatformCVEResolverTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
	ctx      context.Context

	platformCVEView *platformCVEViewMock.MockCveView

	resolver *Resolver
}

func (s *PlatformCVEResolverTestSuite) SetupSuite() {
	s.mockCtrl = gomock.NewController(s.T())
	s.ctx = contextWithClusterPerm(s.T(), s.mockCtrl)
	s.platformCVEView = platformCVEViewMock.NewMockCveView(s.mockCtrl)
	s.resolver, _ = SetupTestResolver(s.T(), s.platformCVEView)
}

func (s *PlatformCVEResolverTestSuite) TearDownSuite() {}

func (s *PlatformCVEResolverTestSuite) TestGetPlatformCVEsNoClusterPerm() {
	q := &PaginatedQuery{}
	response, err := s.resolver.PlatformCVEs(context.Background(), *q)
	s.Error(err)
	s.Nil(response)
}

func (s *PlatformCVEResolverTestSuite) TestGetPlatformCVEsEmpty() {
	q := &PaginatedQuery{}
	expectedQ, err := q.AsV1QueryOrEmpty()
	s.Require().NoError(err)
	expectedQ = tryUnsuppressedQuery(expectedQ)

	s.platformCVEView.EXPECT().Get(s.ctx, expectedQ).Return(nil, nil)
	response, err := s.resolver.PlatformCVEs(s.ctx, *q)
	s.NoError(err)
	s.Len(response, 0)
}

func (s *PlatformCVEResolverTestSuite) TestGetPlatformCVEsNonEmpty() {
	q := &PaginatedQuery{}
	expectedQ, err := q.AsV1QueryOrEmpty()
	s.Require().NoError(err)
	expectedQ = tryUnsuppressedQuery(expectedQ)

	expected := []platformcve.CveCore{
		platformCVEViewMock.NewMockCveCore(s.mockCtrl),
		platformCVEViewMock.NewMockCveCore(s.mockCtrl),
		platformCVEViewMock.NewMockCveCore(s.mockCtrl),
	}

	s.platformCVEView.EXPECT().Get(s.ctx, expectedQ).Return(expected, nil)
	response, err := s.resolver.PlatformCVEs(s.ctx, *q)
	s.NoError(err)
	s.Len(response, 3)
}

func (s *PlatformCVEResolverTestSuite) TestGetPlatformCVEsWithQuery() {
	q := &PaginatedQuery{
		Query: pointers.String("CVE:cve-2022-xyz"),
	}
	expectedQ := search.NewQueryBuilder().AddStrings(search.CVE, "cve-2022-xyz").
		WithPagination(search.NewPagination().Limit(paginated.Unlimited)).ProtoQuery()
	expectedQ = tryUnsuppressedQuery(expectedQ)

	expected := []platformcve.CveCore{
		platformCVEViewMock.NewMockCveCore(s.mockCtrl),
		platformCVEViewMock.NewMockCveCore(s.mockCtrl),
		platformCVEViewMock.NewMockCveCore(s.mockCtrl),
	}

	s.platformCVEView.EXPECT().Get(s.ctx, expectedQ).Return(expected, nil)
	response, err := s.resolver.PlatformCVEs(s.ctx, *q)
	s.NoError(err)
	s.Len(response, 3)
}

func (s *PlatformCVEResolverTestSuite) TestPlatformCVEsCVEsWithPaginatedQuery() {
	q := &PaginatedQuery{
		Pagination: &inputtypes.Pagination{
			SortOption: &inputtypes.SortOption{
				Field: pointers.String(search.CVSS.String()),
				AggregateBy: &inputtypes.AggregateBy{
					AggregateFunc: pointers.String(aggregatefunc.Max.Name()),
				},
			},
		},
	}
	expectedQ := search.NewQueryBuilder().WithPagination(
		search.NewPagination().AddSortOption(
			search.NewSortOption(search.CVSS).AggregateBy(aggregatefunc.Max, false),
		).Limit(paginated.Unlimited),
	).ProtoQuery()
	expectedQ = tryUnsuppressedQuery(expectedQ)

	s.platformCVEView.EXPECT().Get(s.ctx, expectedQ).Return(nil, nil)
	_, err := s.resolver.PlatformCVEs(s.ctx, *q)
	s.NoError(err)
}

func (s *PlatformCVEResolverTestSuite) TestPlatformCVECountNoClusterPerm() {
	response, err := s.resolver.PlatformCVECount(context.Background(), RawQuery{})
	s.Error(err)
	s.Zero(response)
}

func (s *PlatformCVEResolverTestSuite) TestPlatformCVECount() {
	q := &RawQuery{}
	expectedQ, err := q.AsV1QueryOrEmpty()
	s.Require().NoError(err)
	expectedQ = tryUnsuppressedQuery(expectedQ)

	s.platformCVEView.EXPECT().Count(s.ctx, expectedQ).Return(0, nil)
	response, err := s.resolver.PlatformCVECount(s.ctx, *q)
	s.NoError(err)
	s.Equal(response, int32(0))
}

func (s *PlatformCVEResolverTestSuite) TestPlatformCVECountWithQuery() {
	q := &RawQuery{
		Query: pointers.String("Cluster:c1"),
	}
	expectedQ := search.NewQueryBuilder().AddStrings(search.Cluster, "c1").ProtoQuery()
	expectedQ = tryUnsuppressedQuery(expectedQ)

	s.platformCVEView.EXPECT().Count(s.ctx, expectedQ).Return(3, nil)
	response, err := s.resolver.PlatformCVECount(s.ctx, *q)
	s.NoError(err)
	s.Equal(response, int32(3))
}

func (s *PlatformCVEResolverTestSuite) TestGetPlatformCVEMalformed() {
	_, err := s.resolver.PlatformCVE(s.ctx, struct {
		CveID              *string
		SubfieldScopeQuery *string
	}{})
	s.Error(err)
}

func (s *PlatformCVEResolverTestSuite) TestGetPlatformCVENonEmpty() {
	// without filter
	expectedQ := search.NewQueryBuilder().AddExactMatches(search.CVEID, "cve-xyz#K8S_CVE").ProtoQuery()
	expected := []platformcve.CveCore{
		platformCVEViewMock.NewMockCveCore(s.mockCtrl),
	}

	s.platformCVEView.EXPECT().Get(s.ctx, expectedQ).Return(expected, nil)
	response, err := s.resolver.PlatformCVE(
		s.ctx, struct {
			CveID              *string
			SubfieldScopeQuery *string
		}{
			CveID: pointers.String("cve-xyz#K8S_CVE"),
		},
	)
	s.NoError(err)
	s.NotNil(response.data)

	// with filter
	expectedQ = search.NewQueryBuilder().
		AddExactMatches(search.CVEID, "cve-xyz#K8S_CVE").
		AddStrings(search.Cluster, "c1").
		ProtoQuery()
	expected = []platformcve.CveCore{
		platformCVEViewMock.NewMockCveCore(s.mockCtrl),
	}

	s.platformCVEView.EXPECT().Get(s.ctx, expectedQ).Return(expected, nil)
	response, err = s.resolver.PlatformCVE(s.ctx, struct {
		CveID              *string
		SubfieldScopeQuery *string
	}{
		CveID:              pointers.String("cve-xyz#K8S_CVE"),
		SubfieldScopeQuery: pointers.String("Cluster:c1"),
	},
	)
	s.NoError(err)
	s.NotNil(response.data)

	// with filter
	expectedQ = search.NewQueryBuilder().
		AddExactMatches(search.CVEID, "cve-xyz#K8S_CVE").
		AddStrings(search.ClusterPlatformType, storage.ClusterType_KUBERNETES_CLUSTER.String()).
		ProtoQuery()
	expected = []platformcve.CveCore{
		platformCVEViewMock.NewMockCveCore(s.mockCtrl),
	}

	s.platformCVEView.EXPECT().Get(s.ctx, expectedQ).Return(expected, nil)
	response, err = s.resolver.PlatformCVE(s.ctx, struct {
		CveID              *string
		SubfieldScopeQuery *string
	}{
		CveID:              pointers.String("cve-xyz#K8S_CVE"),
		SubfieldScopeQuery: pointers.String("Cluster Platform Type:KUBERNETES_CLUSTER"),
	},
	)
	s.NoError(err)
	s.NotNil(response.data)
}
