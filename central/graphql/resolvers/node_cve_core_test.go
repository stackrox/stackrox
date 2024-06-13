package resolvers

import (
	"context"
	"math"
	"testing"

	"github.com/stackrox/rox/central/graphql/resolvers/inputtypes"
	"github.com/stackrox/rox/central/views/nodecve"
	nodeCVEViewMock "github.com/stackrox/rox/central/views/nodecve/mocks"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestNodeCVECoreResolver(t *testing.T) {
	suite.Run(t, new(NodeCVECoreResolverTestSuite))
}

type NodeCVECoreResolverTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
	ctx      context.Context

	nodeCVEView *nodeCVEViewMock.MockCveView

	resolver *Resolver
}

func (s *NodeCVECoreResolverTestSuite) SetupSuite() {
	s.T().Setenv(features.VulnMgmtNodePlatformCVEs.EnvVar(), "true")

	if !features.VulnMgmtNodePlatformCVEs.Enabled() {
		s.T().Skipf("Skiping test. %s=false", features.VulnMgmtNodePlatformCVEs.EnvVar())
		s.T().SkipNow()
	}

	s.mockCtrl = gomock.NewController(s.T())
	s.ctx = contextWithNodePerm(s.T(), s.mockCtrl)
	s.nodeCVEView = nodeCVEViewMock.NewMockCveView(s.mockCtrl)
	s.resolver, _ = SetupTestResolver(s.T(), s.nodeCVEView)
}

func (s *NodeCVECoreResolverTestSuite) TearDownSuite() {}

func (s *NodeCVECoreResolverTestSuite) TestNodeCVEsEmpty() {
	q := &PaginatedQuery{}
	expectedQ, err := q.AsV1QueryOrEmpty()
	s.Require().NoError(err)

	s.nodeCVEView.EXPECT().Get(s.ctx, expectedQ).Return(nil, nil)
	response, err := s.resolver.NodeCVEs(s.ctx, *q)
	s.NoError(err)
	s.Len(response, 0)
}

func (s *NodeCVECoreResolverTestSuite) TestNodeCVEsNonEmpty() {
	q := &PaginatedQuery{}
	expectedQ, err := q.AsV1QueryOrEmpty()
	s.Require().NoError(err)

	expected := []nodecve.CveCore{
		nodeCVEViewMock.NewMockCveCore(s.mockCtrl),
		nodeCVEViewMock.NewMockCveCore(s.mockCtrl),
		nodeCVEViewMock.NewMockCveCore(s.mockCtrl),
	}

	s.nodeCVEView.EXPECT().Get(s.ctx, expectedQ).Return(expected, nil)
	response, err := s.resolver.NodeCVEs(s.ctx, *q)
	s.NoError(err)
	s.Len(response, 3)
}

func (s *NodeCVECoreResolverTestSuite) TestNodeCVEsWithQuery() {
	q := &PaginatedQuery{
		Query: pointers.String("CVE:cve-2022-xyz"),
	}
	expectedQ := search.NewQueryBuilder().AddStrings(search.CVE, "cve-2022-xyz").
		WithPagination(search.NewPagination().Limit(math.MaxInt32)).ProtoQuery()

	expected := []nodecve.CveCore{
		nodeCVEViewMock.NewMockCveCore(s.mockCtrl),
		nodeCVEViewMock.NewMockCveCore(s.mockCtrl),
		nodeCVEViewMock.NewMockCveCore(s.mockCtrl),
	}

	s.nodeCVEView.EXPECT().Get(s.ctx, expectedQ).Return(expected, nil)
	response, err := s.resolver.NodeCVEs(s.ctx, *q)
	s.NoError(err)
	s.Len(response, 3)
}

func (s *NodeCVECoreResolverTestSuite) TestNodeCVEsWithPaginatedQuery() {
	q := &PaginatedQuery{
		Pagination: &inputtypes.Pagination{
			SortOption: &inputtypes.SortOption{
				Field: pointers.String(search.NodeTopCVSS.String()),
				AggregateBy: &inputtypes.AggregateBy{
					AggregateFunc: pointers.String(aggregatefunc.Max.Name()),
				},
			},
		},
	}
	expectedQ := search.NewQueryBuilder().WithPagination(
		search.NewPagination().AddSortOption(
			search.NewSortOption(search.NodeTopCVSS).AggregateBy(aggregatefunc.Max, false),
		).Limit(math.MaxInt32),
	).ProtoQuery()
	expected := []nodecve.CveCore{
		nodeCVEViewMock.NewMockCveCore(s.mockCtrl),
		nodeCVEViewMock.NewMockCveCore(s.mockCtrl),
		nodeCVEViewMock.NewMockCveCore(s.mockCtrl),
	}

	s.nodeCVEView.EXPECT().Get(s.ctx, expectedQ).Return(expected, nil)
	response, err := s.resolver.NodeCVEs(s.ctx, *q)
	s.NoError(err)
	s.Len(response, 3)
}

func (s *NodeCVECoreResolverTestSuite) TestNodeCVEsNoNodePerm() {
	response, err := s.resolver.NodeCVEs(context.Background(), PaginatedQuery{})
	s.Error(err)
	s.Nil(response)
}

func (s *NodeCVECoreResolverTestSuite) TestNodeCVECountNoNodePerm() {
	response, err := s.resolver.NodeCVECount(context.Background(), RawQuery{})
	s.Error(err)
	s.Zero(response)
}

func (s *NodeCVECoreResolverTestSuite) TestNodeCVECount() {
	q := &RawQuery{}
	expectedQ, err := q.AsV1QueryOrEmpty()
	s.Require().NoError(err)

	s.nodeCVEView.EXPECT().Count(s.ctx, expectedQ).Return(11, nil)
	response, err := s.resolver.NodeCVECount(s.ctx, *q)
	s.NoError(err)
	s.Equal(response, int32(11))
}

func (s *NodeCVECoreResolverTestSuite) TestNodeCVECountWithQuery() {
	q := &RawQuery{
		Query: pointers.String("Node:node"),
	}
	expectedQ := search.NewQueryBuilder().AddStrings(search.Node, "node").ProtoQuery()

	s.nodeCVEView.EXPECT().Count(s.ctx, expectedQ).Return(3, nil)
	response, err := s.resolver.NodeCVECount(s.ctx, *q)
	s.NoError(err)
	s.Equal(response, int32(3))
}

func (s *NodeCVECoreResolverTestSuite) TestNodeCVECountWithInternalError() {
	q := &RawQuery{}
	expectedQ, err := q.AsV1QueryOrEmpty()
	s.Require().NoError(err)

	s.nodeCVEView.EXPECT().Count(s.ctx, expectedQ).Return(0, errox.ServerError)
	response, err := s.resolver.NodeCVECount(s.ctx, *q)
	s.ErrorIs(err, errox.ServerError)
	s.Equal(response, int32(0))
}

func (s *NodeCVECoreResolverTestSuite) TestNodeCVEMalformed() {
	_, err := s.resolver.NodeCVE(s.ctx, struct {
		Cve                *string
		SubfieldScopeQuery *string
	}{})
	s.Error(err)
}

func (s *NodeCVECoreResolverTestSuite) TestNodeCVENoNodePerm() {
	ctx := context.Background()
	response, err := s.resolver.NodeCVE(
		ctx, struct {
			Cve                *string
			SubfieldScopeQuery *string
		}{
			Cve: pointers.String("cve-xyz"),
		},
	)
	s.Error(err)
	s.Nil(response)
}

func (s *NodeCVECoreResolverTestSuite) TestNodeCVENonEmpty() {
	// without filter
	expectedQ := search.NewQueryBuilder().AddExactMatches(search.CVE, "cve-xyz").ProtoQuery()
	expected := []nodecve.CveCore{
		nodeCVEViewMock.NewMockCveCore(s.mockCtrl),
	}

	s.nodeCVEView.EXPECT().Get(s.ctx, expectedQ).Return(expected, nil)
	response, err := s.resolver.NodeCVE(
		s.ctx, struct {
			Cve                *string
			SubfieldScopeQuery *string
		}{
			Cve: pointers.String("cve-xyz"),
		},
	)
	s.NoError(err)
	s.NotNil(response.data)

	// with fixable filter
	expectedQ = search.NewQueryBuilder().
		AddExactMatches(search.CVE, "cve-xyz").
		AddStrings(search.Fixable, "true").
		ProtoQuery()
	expected = []nodecve.CveCore{
		nodeCVEViewMock.NewMockCveCore(s.mockCtrl),
	}

	s.nodeCVEView.EXPECT().Get(s.ctx, expectedQ).Return(expected, nil)
	response, err = s.resolver.NodeCVE(s.ctx, struct {
		Cve                *string
		SubfieldScopeQuery *string
	}{
		Cve:                pointers.String("cve-xyz"),
		SubfieldScopeQuery: pointers.String("Fixable:true"),
	},
	)
	s.NoError(err)
	s.NotNil(response.data)

	// with namespace filter
	expectedQ = search.NewQueryBuilder().
		AddExactMatches(search.CVE, "cve-xyz").
		AddStrings(search.Namespace, "n1").
		ProtoQuery()
	expected = []nodecve.CveCore{
		nodeCVEViewMock.NewMockCveCore(s.mockCtrl),
	}

	s.nodeCVEView.EXPECT().Get(s.ctx, expectedQ).Return(expected, nil)
	response, err = s.resolver.NodeCVE(s.ctx, struct {
		Cve                *string
		SubfieldScopeQuery *string
	}{
		Cve:                pointers.String("cve-xyz"),
		SubfieldScopeQuery: pointers.String("Namespace:n1"),
	},
	)
	s.NoError(err)
	s.NotNil(response.data)
}
