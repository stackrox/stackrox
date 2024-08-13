package search

import (
	"context"
	"testing"

	storeMocks "github.com/stackrox/rox/central/cve/node/datastore/store/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	pkgSearcherMocks "github.com/stackrox/rox/pkg/search/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestNodeCVESearcher(t *testing.T) {
	suite.Run(t, new(NodeCVESearcherSuite))
}

type NodeCVESearcherSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller

	ctx             context.Context
	storage         *storeMocks.MockStore
	embeddedSearher *pkgSearcherMocks.MockSearcher
	searcher        *searcherImpl
}

func (s *NodeCVESearcherSuite) SetupSuite() {
	s.mockCtrl = gomock.NewController(s.T())

	s.ctx = sac.WithAllAccess(context.Background())
	s.storage = storeMocks.NewMockStore(s.mockCtrl)
	s.embeddedSearher = pkgSearcherMocks.NewMockSearcher(s.mockCtrl)
	s.searcher = &searcherImpl{
		storage:  s.storage,
		searcher: s.embeddedSearher,
	}
}

func (s *NodeCVESearcherSuite) TearDownSuite() {
	s.mockCtrl.Finish()
}

func (s *NodeCVESearcherSuite) TestSearchCVEs() {
	cves := getNodeCVEs()

	q := search.EmptyQuery()
	q.Pagination = &v1.QueryPagination{
		Limit: 10,
	}

	// Without orphaned CVEs
	expectedQ := getExpectedQuery(q, false)
	s.embeddedSearher.EXPECT().Search(s.ctx, expectedQ).Times(1).Return([]search.Result{{ID: cves[0].Id}}, nil)
	s.storage.EXPECT().GetMany(s.ctx, []string{cves[0].Id}).Times(1).Return([]*storage.NodeCVE{cves[0]}, []int{}, nil)

	results, err := s.searcher.SearchCVEs(s.ctx, q, false)
	s.Require().NoError(err)
	s.Require().Equal(1, len(results))

	// With orphaned CVEs
	expectedQ = getExpectedQuery(q, true)
	s.embeddedSearher.EXPECT().Search(s.ctx, expectedQ).Times(1).Return([]search.Result{{ID: cves[0].Id}, {ID: cves[1].Id}}, nil)
	s.storage.EXPECT().GetMany(s.ctx, []string{cves[0].Id, cves[1].Id}).Times(1).Return(cves, []int{}, nil)

	results, err = s.searcher.SearchCVEs(s.ctx, q, true)
	s.Require().NoError(err)
	s.Require().Equal(2, len(results))
}

func (s *NodeCVESearcherSuite) TestSearch() {
	cves := getNodeCVEs()

	q := search.EmptyQuery()
	q.Pagination = &v1.QueryPagination{
		Limit: 10,
	}

	// Without orphaned CVEs
	expectedQ := getExpectedQuery(q, false)
	s.embeddedSearher.EXPECT().Search(s.ctx, expectedQ).Times(1).Return([]search.Result{{ID: cves[0].Id}}, nil)

	results, err := s.searcher.Search(s.ctx, q, false)
	s.Require().NoError(err)
	s.Require().Equal(1, len(results))

	// With orphaned CVEs
	expectedQ = getExpectedQuery(q, true)
	s.embeddedSearher.EXPECT().Search(s.ctx, expectedQ).Times(1).Return([]search.Result{{ID: cves[0].Id}, {ID: cves[1].Id}}, nil)

	results, err = s.searcher.Search(s.ctx, q, true)
	s.Require().NoError(err)
	s.Require().Equal(2, len(results))
}

func (s *NodeCVESearcherSuite) TestCount() {
	q := search.EmptyQuery()

	// Without orphaned CVEs
	expectedQ := getExpectedQuery(q, false)
	s.embeddedSearher.EXPECT().Count(s.ctx, expectedQ).Times(1).Return(1, nil)

	count, err := s.searcher.Count(s.ctx, q, false)
	s.Require().NoError(err)
	s.Require().Equal(1, count)

	// With orphaned CVEs
	expectedQ = getExpectedQuery(q, true)
	s.embeddedSearher.EXPECT().Count(s.ctx, expectedQ).Times(1).Return(2, nil)

	count, err = s.searcher.Count(s.ctx, q, true)
	s.Require().NoError(err)
	s.Require().Equal(2, count)
}

func (s *NodeCVESearcherSuite) TestSearchRawCVEs() {
	cves := getNodeCVEs()

	q := search.EmptyQuery()
	q.Pagination = &v1.QueryPagination{
		Limit: 10,
	}

	// Without orphaned CVEs
	expectedQ := getExpectedQuery(q, false)
	s.embeddedSearher.EXPECT().Search(s.ctx, expectedQ).Times(1).Return([]search.Result{{ID: cves[0].Id}}, nil)
	s.storage.EXPECT().GetMany(s.ctx, []string{cves[0].Id}).Times(1).Return([]*storage.NodeCVE{cves[0]}, []int{}, nil)

	results, err := s.searcher.SearchRawCVEs(s.ctx, q, false)
	s.Require().NoError(err)
	s.Require().Equal(1, len(results))

	// With orphaned CVEs
	expectedQ = getExpectedQuery(q, true)
	s.embeddedSearher.EXPECT().Search(s.ctx, expectedQ).Times(1).Return([]search.Result{{ID: cves[0].Id}, {ID: cves[1].Id}}, nil)
	s.storage.EXPECT().GetMany(s.ctx, []string{cves[0].Id, cves[1].Id}).Times(1).Return(cves, []int{}, nil)

	results, err = s.searcher.SearchRawCVEs(s.ctx, q, true)
	s.Require().NoError(err)
	s.Require().Equal(2, len(results))
}

func getExpectedQuery(q *v1.Query, allowOrphaned bool) *v1.Query {
	if allowOrphaned {
		return q
	}
	ret := search.ConjunctionQuery(q, search.NewQueryBuilder().AddBools(search.CVEOrphaned, false).ProtoQuery())
	ret.Pagination = q.GetPagination()
	return ret
}

func getNodeCVEs() []*storage.NodeCVE {
	return []*storage.NodeCVE{
		{
			Id: "CVE-123-456#rhel9",
			CveBaseInfo: &storage.CVEInfo{
				Cve: "CVE-123-456",
			},
			Orphaned: false,
		},
		{
			Id: "CVE-234-567#rhel9",
			CveBaseInfo: &storage.CVEInfo{
				Cve: "CVE-234-567",
			},
			Orphaned: true,
		},
	}
}
