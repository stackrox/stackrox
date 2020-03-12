package filtered

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/dackbox/graph/mocks"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	searchMocks "github.com/stackrox/rox/pkg/search/blevesearch/mocks"
	"github.com/stretchr/testify/suite"
)

var (
	prefix1 = []byte("pre1")
	prefix2 = []byte("pre2")
	prefix3 = []byte("namespacesSACBucket")
	prefix4 = []byte("clusters")

	id1 = []byte("id1")
	id2 = []byte("id2")
	id3 = []byte("id3")
	id4 = []byte("id4")
	id5 = []byte("id5")
	id6 = []byte("id6")
	id7 = []byte("id7")
	id8 = []byte("id8")
	id9 = []byte("id9")

	prefixedID1 = badgerhelper.GetBucketKey(prefix1, id1)
	prefixedID2 = badgerhelper.GetBucketKey(prefix1, id2)
	prefixedID3 = badgerhelper.GetBucketKey(prefix2, id3)
	prefixedID4 = badgerhelper.GetBucketKey(prefix2, id4)
	prefixedID9 = badgerhelper.GetBucketKey(prefix2, id9)
	prefixedID5 = badgerhelper.GetBucketKey(prefix3, id5)
	prefixedID6 = badgerhelper.GetBucketKey(prefix3, id6)
	prefixedID7 = badgerhelper.GetBucketKey(prefix4, id7)
	prefixedID8 = badgerhelper.GetBucketKey(prefix4, id8)

	// id1 -> id9
	//      \ id3 -> id5 (namespace) -> id7, id8 (cluster)
	// id2 -> id4 -> id6 (namespace) -> id7 (cluster)
	toID1 = [][]byte{prefixedID9, prefixedID3}
	toID2 = [][]byte{prefixedID4}
	toID3 = [][]byte{prefixedID5}
	toID4 = [][]byte{prefixedID6}
	toID5 = [][]byte{prefixedID7, prefixedID8}
	toID6 = [][]byte{prefixedID7}

	globalResource = permissions.ResourceMetadata{
		Resource: "resource",
		Scope:    permissions.GlobalScope,
	}
	clusterResource = permissions.ResourceMetadata{
		Resource: "resource",
		Scope:    permissions.ClusterScope,
	}
	namespaceResource = permissions.ResourceMetadata{
		Resource: "resource",
		Scope:    permissions.NamespaceScope,
	}
)

func TestFilteredSearcher(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(filteredSearcherTestSuite))
}

type filteredSearcherTestSuite struct {
	suite.Suite

	mockRGraph         *mocks.MockRGraph
	mockUnsafeSearcher *searchMocks.MockUnsafeSearcher

	mockCtrl *gomock.Controller
}

func (s *filteredSearcherTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockRGraph = mocks.NewMockRGraph(s.mockCtrl)
	s.mockUnsafeSearcher = searchMocks.NewMockUnsafeSearcher(s.mockCtrl)
}

func (s *filteredSearcherTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *filteredSearcherTestSuite) TestGlobalAllowed() {
	s.mockUnsafeSearcher.EXPECT().Search(gomock.Any()).Return([]search.Result{
		{
			ID: string(id1),
		},
		{
			ID: string(id2),
		},
	}, nil)

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())

	filter, err := NewSACFilter(
		WithResourceHelper(sac.ForResource(globalResource)),
	)
	s.NoError(err, "filter creation should have succeeded")

	searcher := UnsafeSearcher(s.mockUnsafeSearcher, filter)
	results, err := searcher.Search(ctx, &v1.Query{})
	s.NoError(err, "search should have succeeded")
	s.Equal([]search.Result{
		{
			ID: string(id1),
		},
		{
			ID: string(id2),
		},
	}, results)
}

func (s *filteredSearcherTestSuite) TestGlobalDenied() {
	s.mockUnsafeSearcher.EXPECT().Search(gomock.Any()).Return([]search.Result{
		{
			ID: string(id1),
		},
		{
			ID: string(id2),
		},
	}, nil)

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())

	filter, err := NewSACFilter(
		WithResourceHelper(sac.ForResource(globalResource)),
	)
	s.NoError(err, "filter creation should have succeeded")

	searcher := UnsafeSearcher(s.mockUnsafeSearcher, filter)
	results, err := searcher.Search(ctx, &v1.Query{})
	s.NoError(err, "search should have succeeded")
	s.Equal([]search.Result{}, results)
}

func (s *filteredSearcherTestSuite) TestClusterScoped() {
	// Expect graph and search interactions.
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID1).Return(toID1)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID2).Return(toID2)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID3).Return(toID3)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID4).Return(toID4)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID5).Return(toID5)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID6).Return(toID6)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID9).Return(nil)

	s.mockUnsafeSearcher.EXPECT().Search(gomock.Any()).Return([]search.Result{
		{
			ID: string(id1),
		},
		{
			ID: string(id2),
		},
	}, nil)

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
		sac.ResourceScopeKeys(clusterResource),
		sac.ClusterScopeKeys("id7")))

	filter, err := NewSACFilter(
		WithResourceHelper(sac.ForResource(clusterResource)),
		WithGraphProvider(fakeGraphProvider{mg: s.mockRGraph}),
		WithClusterPath(prefix1, prefix2, prefix3, prefix4),
	)
	s.NoError(err, "filter creation should have succeeded")

	searcher := UnsafeSearcher(s.mockUnsafeSearcher, filter)
	results, err := searcher.Search(ctx, &v1.Query{})
	s.NoError(err, "search should have succeeded")
	s.Equal([]search.Result{
		{
			ID: string(id1),
		},
		{
			ID: string(id2),
		},
	}, results)
}

func (s *filteredSearcherTestSuite) TestClusterScopedMultiCluster() {
	// Expect graph and search interactions.
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID1).Return(toID1)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID2).Return(toID2)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID3).Return(toID3)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID4).Return(toID4)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID5).Return(toID5)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID6).Return(toID6)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID9).Return(nil)

	s.mockUnsafeSearcher.EXPECT().Search(gomock.Any()).Return([]search.Result{
		{
			ID: string(id1),
		},
		{
			ID: string(id2),
		},
	}, nil)

	// Allow first namespace and cluster
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
		sac.ResourceScopeKeys(clusterResource),
		sac.ClusterScopeKeys("id8")))

	filter, err := NewSACFilter(
		WithResourceHelper(sac.ForResource(clusterResource)),
		WithGraphProvider(fakeGraphProvider{mg: s.mockRGraph}),
		WithClusterPath(prefix1, prefix2, prefix3, prefix4),
	)
	s.NoError(err, "filter creation should have succeeded")

	searcher := UnsafeSearcher(s.mockUnsafeSearcher, filter)
	results, err := searcher.Search(ctx, &v1.Query{})
	s.NoError(err, "search should have succeeded")
	s.Equal([]search.Result{
		{
			ID: string(id1),
		},
	}, results)
}

func (s *filteredSearcherTestSuite) TestNamespaceScoped() {
	// Expect graph and search interactions.
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID1).Return(toID1)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID2).Return(toID2)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID3).Return(toID3)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID4).Return(toID4)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID5).Return(toID5)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID6).Return(toID6)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID9).Return(nil)

	s.mockUnsafeSearcher.EXPECT().Search(gomock.Any()).Return([]search.Result{
		{
			ID: string(id1),
		},
		{
			ID: string(id2),
		},
	}, nil)

	// Allow first namespace and cluster
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
		sac.ResourceScopeKeys(namespaceResource),
		sac.ClusterScopeKeys("id7"),
		sac.NamespaceScopeKeys("id5")))

	filter, err := NewSACFilter(
		WithResourceHelper(sac.ForResource(namespaceResource)),
		WithGraphProvider(fakeGraphProvider{mg: s.mockRGraph}),
		WithNamespacePath(prefix1, prefix2, prefix3),
		WithClusterPath(prefix1, prefix2, prefix3, prefix4),
	)
	s.NoError(err, "filter creation should have succeeded")

	searcher := UnsafeSearcher(s.mockUnsafeSearcher, filter)
	results, err := searcher.Search(ctx, &v1.Query{})
	s.NoError(err, "search should have succeeded")
	s.Equal([]search.Result{
		{
			ID: string(id1),
		},
	}, results)
}

func (s *filteredSearcherTestSuite) TestNamespaceScopedMultiCluster() {
	// Expect graph and search interactions.
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID1).Return(toID1)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID2).Return(toID2)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID3).Return(toID3)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID4).Return(toID4)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID5).Return(toID5)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID6).Return(toID6)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID9).Return(nil)

	s.mockUnsafeSearcher.EXPECT().Search(gomock.Any()).Return([]search.Result{
		{
			ID: string(id1),
		},
		{
			ID: string(id2),
		},
	}, nil)

	// Allow first namespace and cluster
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
		sac.ResourceScopeKeys(namespaceResource),
		sac.ClusterScopeKeys("id8")))

	filter, err := NewSACFilter(
		WithResourceHelper(sac.ForResource(namespaceResource)),
		WithGraphProvider(fakeGraphProvider{mg: s.mockRGraph}),
		WithNamespacePath(prefix1, prefix2, prefix3),
		WithClusterPath(prefix1, prefix2, prefix3, prefix4),
	)
	s.NoError(err, "filter creation should have succeeded")

	searcher := UnsafeSearcher(s.mockUnsafeSearcher, filter)
	results, err := searcher.Search(ctx, &v1.Query{})
	s.NoError(err, "search should have succeeded")
	s.Equal([]search.Result{
		{
			ID: string(id1),
		},
	}, results)
}

func (s *filteredSearcherTestSuite) TestMutipleSACFiltersFail() {
	// Expect graph and search interactions.
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID1).Return(toID1)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID2).Return(toID2)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID3).Return(toID3)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID4).Return(toID4)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID9).Return(nil)

	s.mockUnsafeSearcher.EXPECT().Search(gomock.Any()).Return([]search.Result{
		{
			ID: string(id1),
		},
		{
			ID: string(id2),
		},
	}, nil)

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
		sac.ResourceScopeKeys(clusterResource),
		sac.ClusterScopeKeys("id7")))

	filter1, err := NewSACFilter(
		WithResourceHelper(sac.ForResource(clusterResource)),
		WithGraphProvider(fakeGraphProvider{mg: s.mockRGraph}),
		WithClusterPath(prefix1, prefix2, prefix3),
	)
	s.NoError(err, "filter creation should have succeeded")

	searcher := UnsafeSearcher(s.mockUnsafeSearcher, filter1)
	results, err := searcher.Search(ctx, &v1.Query{})
	s.Nil(err, "filtered searcher should have succeeded")
	s.Len(results, 0, "filtered results should have been empty")
}

func (s *filteredSearcherTestSuite) TestMutipleSACFiltersPass() {
	// Expect graph and search interactions.
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID1).Return(toID1)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID2).Return(toID2)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID3).Return(toID3)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID4).Return(toID4)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID9).Return(nil)

	s.mockRGraph.EXPECT().GetRefsTo(prefixedID1).Return(toID1)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID2).Return(toID2)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID3).Return(toID3)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID4).Return(toID4)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID5).Return(toID5)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID6).Return(toID6)
	s.mockRGraph.EXPECT().GetRefsTo(prefixedID9).Return(nil)

	s.mockUnsafeSearcher.EXPECT().Search(gomock.Any()).Return([]search.Result{
		{
			ID: string(id1),
		},
		{
			ID: string(id2),
		},
	}, nil)

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
		sac.ResourceScopeKeys(clusterResource),
		sac.ClusterScopeKeys("id7")))

	filter1, err := NewSACFilter(
		WithResourceHelper(sac.ForResource(clusterResource)),
		WithGraphProvider(fakeGraphProvider{mg: s.mockRGraph}),
		WithClusterPath(prefix1, prefix2, prefix3),
	)
	s.NoError(err, "filter creation should have succeeded")

	filter2, err := NewSACFilter(
		WithResourceHelper(sac.ForResource(clusterResource)),
		WithGraphProvider(fakeGraphProvider{mg: s.mockRGraph}),
		WithClusterPath(prefix1, prefix2, prefix3, prefix4),
	)
	s.NoError(err, "filter creation should have succeeded")

	searcher := UnsafeSearcher(s.mockUnsafeSearcher, []Filter{filter1, filter2}...)
	results, err := searcher.Search(ctx, &v1.Query{})
	s.NoError(err, "search should have succeeded")
	s.Equal([]search.Result{
		{
			ID: string(id1),
		},
		{
			ID: string(id2),
		},
	}, results)
}

type fakeGraphProvider struct {
	mg *mocks.MockRGraph
}

func (fgp fakeGraphProvider) NewGraphView() graph.DiscardableRGraph {
	return graph.NewDiscardableGraph(fgp.mg, func() {})
}
