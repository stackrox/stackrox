package resolvers

import (
	"context"
	"math"
	"testing"
	"time"

	nodeCVEMocks "github.com/stackrox/rox/central/cve/node/datastore/mocks"
	"github.com/stackrox/rox/central/graphql/resolvers/inputtypes"
	nodeMocks "github.com/stackrox/rox/central/node/datastore/mocks"
	"github.com/stackrox/rox/central/views/nodecve"
	nodeCVEViewMock "github.com/stackrox/rox/central/views/nodecve/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
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

	nodeCVEDatastore *nodeCVEMocks.MockDataStore
	nodeDatastore    *nodeMocks.MockDataStore
	nodeCVEView      *nodeCVEViewMock.MockCveView

	resolver *Resolver
}

func (s *NodeCVECoreResolverTestSuite) SetupSuite() {
	s.mockCtrl = gomock.NewController(s.T())
	s.ctx = contextWithNodePerm(s.T(), s.mockCtrl)
	s.nodeCVEView = nodeCVEViewMock.NewMockCveView(s.mockCtrl)
	s.nodeCVEDatastore = nodeCVEMocks.NewMockDataStore(s.mockCtrl)
	s.nodeDatastore = nodeMocks.NewMockDataStore(s.mockCtrl)
	s.resolver, _ = SetupTestResolver(s.T(), s.nodeCVEView, s.nodeCVEDatastore, s.nodeDatastore)
}

func (s *NodeCVECoreResolverTestSuite) TearDownSuite() {}

func (s *NodeCVECoreResolverTestSuite) TestNodeCVEsEmpty() {
	q := &PaginatedQuery{}
	expectedQ, err := q.AsV1QueryOrEmpty()
	s.Require().NoError(err)
	expectedQ = tryUnsuppressedQuery(expectedQ)
	expectedQ = noOrphanedCVEsQuery(s.T(), expectedQ)

	s.nodeCVEView.EXPECT().Get(s.ctx, expectedQ).Return(nil, nil)
	response, err := s.resolver.NodeCVEs(s.ctx, *q)
	s.NoError(err)
	s.Len(response, 0)
}

func (s *NodeCVECoreResolverTestSuite) TestNodeCVEsNonEmpty() {
	q := &PaginatedQuery{}
	expectedQ, err := q.AsV1QueryOrEmpty()
	s.Require().NoError(err)
	expectedQ = tryUnsuppressedQuery(expectedQ)
	expectedQ = noOrphanedCVEsQuery(s.T(), expectedQ)

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
		WithPagination(search.NewPagination().Limit(paginated.Unlimited)).ProtoQuery()
	expectedQ = tryUnsuppressedQuery(expectedQ)
	expectedQ = noOrphanedCVEsQuery(s.T(), expectedQ)

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
		).Limit(paginated.Unlimited),
	).ProtoQuery()
	expectedQ = tryUnsuppressedQuery(expectedQ)
	expectedQ = noOrphanedCVEsQuery(s.T(), expectedQ)
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
	expectedQ = tryUnsuppressedQuery(expectedQ)
	expectedQ = noOrphanedCVEsQuery(s.T(), expectedQ)

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
	expectedQ = tryUnsuppressedQuery(expectedQ)
	expectedQ = noOrphanedCVEsQuery(s.T(), expectedQ)

	s.nodeCVEView.EXPECT().Count(s.ctx, expectedQ).Return(3, nil)
	response, err := s.resolver.NodeCVECount(s.ctx, *q)
	s.NoError(err)
	s.Equal(response, int32(3))
}

func (s *NodeCVECoreResolverTestSuite) TestNodeCVECountWithInternalError() {
	q := &RawQuery{}
	expectedQ, err := q.AsV1QueryOrEmpty()
	s.Require().NoError(err)
	expectedQ = tryUnsuppressedQuery(expectedQ)
	expectedQ = noOrphanedCVEsQuery(s.T(), expectedQ)

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
	expectedQ = noOrphanedCVEsQuery(s.T(), expectedQ)
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
	expectedQ = noOrphanedCVEsQuery(s.T(), expectedQ)
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
	expectedQ = noOrphanedCVEsQuery(s.T(), expectedQ)
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

func (s *NodeCVECoreResolverTestSuite) TestNodeCVECountBySeverity() {
	q := &RawQuery{}
	expectedQ, err := q.AsV1QueryOrEmpty()
	s.Require().NoError(err)
	expectedQ = tryUnsuppressedQuery(expectedQ)
	expectedQ = noOrphanedCVEsQuery(s.T(), expectedQ)
	cbs := nodecve.NewCountByNodeCVESeverity(7, 3, 6, 2, 5, 1, 4, 0, 0, 0)

	s.nodeCVEView.EXPECT().CountBySeverity(s.ctx, expectedQ).Return(cbs, nil)
	response, err := s.resolver.NodeCVECountBySeverity(s.ctx, *q)
	s.NoError(err)

	critical, err := response.Critical(s.ctx)
	s.NoError(err)
	s.Equal(int32(7), critical.Total(s.ctx))
	s.Equal(int32(3), critical.Fixable(s.ctx))

	important, err := response.Important(s.ctx)
	s.NoError(err)
	s.Equal(int32(6), important.Total(s.ctx))
	s.Equal(int32(2), important.Fixable(s.ctx))

	moderate, err := response.Moderate(s.ctx)
	s.NoError(err)
	s.Equal(int32(5), moderate.Total(s.ctx))
	s.Equal(int32(1), moderate.Fixable(s.ctx))

	low, err := response.Low(s.ctx)
	s.NoError(err)
	s.Equal(int32(4), low.Total(s.ctx))
	s.Equal(int32(0), low.Fixable(s.ctx))
}

func (s *NodeCVECoreResolverTestSuite) TestNodeCVECountBySeverityWithQuery() {
	q := &RawQuery{
		Query: pointers.String("Node:node"),
	}
	expectedQ := search.NewQueryBuilder().AddStrings(search.Node, "node").ProtoQuery()
	expectedQ = tryUnsuppressedQuery(expectedQ)
	expectedQ = noOrphanedCVEsQuery(s.T(), expectedQ)
	cbs := nodecve.NewCountByNodeCVESeverity(7, 3, 6, 2, 5, 1, 4, 0, 0, 0)

	s.nodeCVEView.EXPECT().CountBySeverity(s.ctx, expectedQ).Return(cbs, nil)
	response, err := s.resolver.NodeCVECountBySeverity(s.ctx, *q)
	s.NoError(err)

	critical, err := response.Critical(s.ctx)
	s.NoError(err)
	s.Equal(int32(7), critical.Total(s.ctx))
	s.Equal(int32(3), critical.Fixable(s.ctx))

	important, err := response.Important(s.ctx)
	s.NoError(err)
	s.Equal(int32(6), important.Total(s.ctx))
	s.Equal(int32(2), important.Fixable(s.ctx))

	moderate, err := response.Moderate(s.ctx)
	s.NoError(err)
	s.Equal(int32(5), moderate.Total(s.ctx))
	s.Equal(int32(1), moderate.Fixable(s.ctx))

	low, err := response.Low(s.ctx)
	s.NoError(err)
	s.Equal(int32(4), low.Total(s.ctx))
	s.Equal(int32(0), low.Fixable(s.ctx))
}

func (s *NodeCVECoreResolverTestSuite) TestNodeCVECountBySeverityWithInternalError() {
	q := &RawQuery{}
	expectedQ, err := q.AsV1QueryOrEmpty()
	s.Require().NoError(err)
	expectedQ = tryUnsuppressedQuery(expectedQ)
	expectedQ = noOrphanedCVEsQuery(s.T(), expectedQ)

	s.nodeCVEView.EXPECT().CountBySeverity(s.ctx, expectedQ).Return(nil, errox.ServerError)
	response, err := s.resolver.NodeCVECountBySeverity(s.ctx, *q)
	s.ErrorIs(err, errox.ServerError)
	s.Nil(response)
}

func (s *NodeCVECoreResolverTestSuite) TestNodeCVECountBySeverityNoNodePerm() {
	response, err := s.resolver.NodeCVECountBySeverity(context.Background(), RawQuery{})
	s.Error(err)
	s.Nil(response)
}

func (s *NodeCVECoreResolverTestSuite) TestNodeCVESubResolvers() {
	// without filter
	cve := "cve-xyz"
	expectedQ := search.NewQueryBuilder().AddExactMatches(search.CVE, cve).ProtoQuery()
	expectedQ = noOrphanedCVEsQuery(s.T(), expectedQ)
	cveCoreMock := nodeCVEViewMock.NewMockCveCore(s.mockCtrl)
	expected := []nodecve.CveCore{
		cveCoreMock,
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

	// CVE
	cveCoreMock.EXPECT().GetCVE().Return(cve).AnyTimes()
	s.Equal(cve, response.CVE(s.ctx))

	// NodeCount
	cveCoreMock.EXPECT().GetNodeCount().Return(3)
	s.Equal(int32(3), response.AffectedNodeCount(s.ctx))

	// OperatingSystemCount
	cveCoreMock.EXPECT().GetOperatingSystemCount().Return(1)
	s.Equal(1, cveCoreMock.GetOperatingSystemCount())

	// TopCVSS
	cveCoreMock.EXPECT().GetTopCVSS().Return(float32(5.5))
	s.Equal(5.5, response.TopCVSS(s.ctx))

	// FirstDiscoveredInSystem
	ts := time.Now()
	cveCoreMock.EXPECT().GetFirstDiscoveredInSystem().Return(&ts)
	s.Equal(ts, response.FirstDiscoveredInSystem(s.ctx).Time)

	// CountByNodeCVESeverity
	sev := nodecve.NewCountByNodeCVESeverity(7, 3, 6, 2, 5, 1, 4, 0, 0, 0)
	cveCoreMock.EXPECT().GetNodeCountBySeverity().Return(sev)
	sevResolver, err := response.AffectedNodeCountBySeverity(s.ctx)
	s.NoError(err)

	critical, err := sevResolver.Critical(s.ctx)
	s.NoError(err)
	s.Equal(int32(sev.GetCriticalSeverityCount().GetTotal()), critical.Total(s.ctx))
	s.Equal(int32(sev.GetCriticalSeverityCount().GetFixable()), critical.Fixable(s.ctx))

	important, err := sevResolver.Important(s.ctx)
	s.NoError(err)
	s.Equal(int32(sev.GetImportantSeverityCount().GetTotal()), important.Total(s.ctx))
	s.Equal(int32(sev.GetImportantSeverityCount().GetFixable()), important.Fixable(s.ctx))

	moderate, err := sevResolver.Moderate(s.ctx)
	s.NoError(err)
	s.Equal(int32(sev.GetModerateSeverityCount().GetTotal()), moderate.Total(s.ctx))
	s.Equal(int32(sev.GetModerateSeverityCount().GetFixable()), moderate.Fixable(s.ctx))

	low, err := sevResolver.Low(s.ctx)
	s.NoError(err)
	s.Equal(int32(sev.GetLowSeverityCount().GetTotal()), low.Total(s.ctx))
	s.Equal(int32(sev.GetLowSeverityCount().GetFixable()), low.Fixable(s.ctx))

	// DistroTuples
	cveIDsToTest := []string{"11", "22"}
	cveResults := []search.Result{
		{
			ID: cveIDsToTest[0],
		},
		{
			ID: cveIDsToTest[1],
		},
	}
	nodeCVEs := []*storage.NodeCVE{
		{
			Id:          cveIDsToTest[0],
			CveBaseInfo: &storage.CVEInfo{Cve: cve},
		},
		{
			Id:          "22",
			CveBaseInfo: &storage.CVEInfo{Cve: cve},
		},
	}
	cveCoreMock.EXPECT().GetCVEIDs().Return(cveIDsToTest)
	expectedQ = search.NewQueryBuilder().AddExactMatches(search.CVEID, cveIDsToTest...).
		AddBools(search.CVESuppressed, true, false).
		WithPagination(search.NewPagination().Limit(paginated.Unlimited)).ProtoQuery()
	expectedQ = noOrphanedCVEsQuery(s.T(), expectedQ)

	s.nodeCVEDatastore.EXPECT().Search(s.ctx, expectedQ).Return(cveResults, nil)
	s.nodeCVEDatastore.EXPECT().GetBatch(s.ctx, cveIDsToTest).Return(nodeCVEs, nil)
	vulns, err := response.DistroTuples(s.ctx)
	s.Nil(err)
	s.Len(vulns, 2)
	for _, vuln := range vulns {
		s.Contains(cveIDsToTest, string(vuln.Id(s.ctx)))
		s.Equal(cve, vuln.CVE(s.ctx))
	}

	// Nodes
	nodeIDsToTest := []string{fixtureconsts.Node1, fixtureconsts.Node2}
	expectedNodes := []*storage.Node{
		{
			Id: nodeIDsToTest[0],
		},
		{
			Id: nodeIDsToTest[1],
		},
	}

	expectedQ = search.NewQueryBuilder().AddExactMatches(search.CVE, cve).ProtoQuery()
	expectedQ = search.ConjunctionQuery(expectedQ, response.subFieldQuery)
	s.nodeDatastore.EXPECT().SearchRawNodes(s.ctx, expectedQ).Return(expectedNodes, nil)
	nodes, err := response.Nodes(s.ctx, struct{ Pagination *inputtypes.Pagination }{})
	s.Nil(err)
	s.Len(nodes, 2)
	s.Contains(nodeIDsToTest, string(nodes[0].Id(s.ctx)), string(nodes[1].Id(s.ctx)))
}

func noOrphanedCVEsQuery(_ *testing.T, q *v1.Query) *v1.Query {
	pagination := q.GetPagination()
	ret := search.ConjunctionQuery(q, search.NewQueryBuilder().AddBools(search.CVEOrphaned, false).ProtoQuery())
	ret.Pagination = pagination
	return ret
}
