package filtered

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	searchMocks "github.com/stackrox/rox/pkg/search/blevesearch/mocks"
	"github.com/stretchr/testify/suite"
)

var (
	id1 = "id1"
	id2 = "id2"
	id3 = "id3"
	id4 = "id4"

	cluster1 = "c1"
	cluster2 = "c2"

	namespace1 = "ns1"
	namespace2 = "ns2"
	namespace3 = "ns3"

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

	mockUnsafeSearcher *searchMocks.MockUnsafeSearcher

	mockCtrl *gomock.Controller
}

func (s *filteredSearcherTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockUnsafeSearcher = searchMocks.NewMockUnsafeSearcher(s.mockCtrl)
}

func (s *filteredSearcherTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *filteredSearcherTestSuite) TestGlobalAllowed() {
	s.mockUnsafeSearcher.EXPECT().Search(gomock.Any()).Return([]search.Result{
		{
			ID: id1,
		},
		{
			ID: id2,
		},
		{
			ID: id3,
		},
		{
			ID: id4,
		},
	}, nil)

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())

	filter, err := NewSACFilter(
		WithResourceHelper(sac.ForResource(globalResource)),
		WithReadAccess(),
	)
	s.NoError(err, "filter creation should have succeeded")

	searcher := UnsafeSearcher(s.mockUnsafeSearcher, filter)
	results, err := searcher.Search(ctx, &v1.Query{})
	s.NoError(err, "search should have succeeded")
	s.Equal([]search.Result{
		{
			ID: id1,
		},
		{
			ID: id2,
		},
		{
			ID: id3,
		},
		{
			ID: id4,
		},
	}, results)
}

func (s *filteredSearcherTestSuite) TestGlobalDenied() {
	s.mockUnsafeSearcher.EXPECT().Search(gomock.Any()).Return([]search.Result{
		{
			ID: id1,
		},
		{
			ID: id2,
		},
		{
			ID: id3,
		},
		{
			ID: id4,
		},
	}, nil)

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())

	filter, err := NewSACFilter(
		WithResourceHelper(sac.ForResource(globalResource)),
		WithReadAccess(),
	)
	s.NoError(err, "filter creation should have succeeded")

	searcher := UnsafeSearcher(s.mockUnsafeSearcher, filter)
	results, err := searcher.Search(ctx, &v1.Query{})
	s.NoError(err, "search should have succeeded")
	s.Equal([]search.Result{}, results)
}

func (s *filteredSearcherTestSuite) TestScoped() {
	s.mockUnsafeSearcher.EXPECT().Search(gomock.Any()).Return([]search.Result{
		{
			ID: id1,
		},
		{
			ID: id2,
		},
		{
			ID: id3,
		},
		{
			ID: id4,
		},
	}, nil)

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
		sac.ResourceScopeKeys(clusterResource),
		sac.ClusterScopeKeys(cluster1)))

	filter, err := NewSACFilter(
		WithResourceHelper(sac.ForResource(clusterResource)),
		WithScopeTransform(s.fakeTransformer()),
		WithReadAccess(),
	)
	s.NoError(err, "filter creation should have succeeded")

	searcher := UnsafeSearcher(s.mockUnsafeSearcher, filter)
	results, err := searcher.Search(ctx, &v1.Query{})
	s.NoError(err, "search should have succeeded")
	s.Equal([]search.Result{
		// id1 and id3 are the only ids in cluster1
		{
			ID: id1,
		},
		{
			ID: id3,
		},
	}, results)
}

func (s *filteredSearcherTestSuite) TestMultiScoped() {
	s.mockUnsafeSearcher.EXPECT().Search(gomock.Any()).Return([]search.Result{
		{
			ID: id1,
		},
		{
			ID: id2,
		},
		{
			ID: id3,
		},
		{
			ID: id4,
		},
	}, nil)

	// Allow first namespace and cluster
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
		sac.ResourceScopeKeys(namespaceResource),
		sac.ClusterScopeKeys(cluster2),
		sac.NamespaceScopeKeys(namespace2)))

	filter, err := NewSACFilter(
		WithResourceHelper(sac.ForResource(namespaceResource)),
		WithScopeTransform(s.fakeTransformer()),
		WithReadAccess(),
	)
	s.NoError(err, "filter creation should have succeeded")

	searcher := UnsafeSearcher(s.mockUnsafeSearcher, filter)
	results, err := searcher.Search(ctx, &v1.Query{})
	s.NoError(err, "search should have succeeded")
	s.Equal([]search.Result{
		// Only id2 and id3 are allowed since they are the only ids in cluster2:namespace2
		{
			ID: id2,
		},
		{
			ID: id3,
		},
	}, results)
}

func (s *filteredSearcherTestSuite) TestMutipleSACFiltersFail() {
	s.mockUnsafeSearcher.EXPECT().Search(gomock.Any()).Return([]search.Result{
		{
			ID: id1,
		},
		{
			ID: id2,
		},
		{
			ID: id3,
		},
		{
			ID: id4,
		},
	}, nil)

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
		sac.ResourceScopeKeys(clusterResource),
		sac.ClusterScopeKeys(cluster2)))

	filter1, err := NewSACFilter(
		WithResourceHelper(sac.ForResource(clusterResource)),
		WithReadAccess(),
	)
	s.NoError(err, "filter creation should have succeeded")

	filter2, err := NewSACFilter(
		WithResourceHelper(sac.ForResource(clusterResource)),
		WithReadAccess(),
	)
	s.NoError(err, "filter creation should have succeeded")

	searcher := UnsafeSearcher(s.mockUnsafeSearcher, filter1, filter2)
	results, err := searcher.Search(ctx, &v1.Query{})
	s.Nil(err, "filtered searcher should have succeeded")
	s.Len(results, 0, "filtered results should have been empty")
}

func (s *filteredSearcherTestSuite) TestMutipleSACFiltersPass() {
	s.mockUnsafeSearcher.EXPECT().Search(gomock.Any()).Return([]search.Result{
		{
			ID: id1,
		},
		{
			ID: id2,
		},
		{
			ID: id3,
		},
		{
			ID: id4,
		},
	}, nil)

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
		sac.ResourceScopeKeys(clusterResource),
		sac.ClusterScopeKeys(cluster1, cluster2)))

	filter1, err := NewSACFilter(
		WithResourceHelper(sac.ForResource(clusterResource)),
		WithScopeTransform(s.fakeTransformer()),
		WithReadAccess(),
	)
	s.NoError(err, "filter creation should have succeeded")

	filter2, err := NewSACFilter(
		WithResourceHelper(sac.ForResource(clusterResource)),
		WithScopeTransform(s.fakeTransformer()),
		WithReadAccess(),
	)
	s.NoError(err, "filter creation should have succeeded")

	searcher := UnsafeSearcher(s.mockUnsafeSearcher, filter1, filter2)
	results, err := searcher.Search(ctx, &v1.Query{})
	s.NoError(err, "search should have succeeded")
	s.Equal([]search.Result{
		// All ids present since all clusters are allowed.
		{
			ID: id1,
		},
		{
			ID: id2,
		},
		{
			ID: id3,
		},
		{
			ID: id4,
		},
	}, results)
}

func (s *filteredSearcherTestSuite) fakeTransformer() ScopeTransform {
	return func(_ context.Context, key []byte) [][]sac.ScopeKey {
		sKey := string(key)
		if sKey == id1 {
			return [][]sac.ScopeKey{{sac.ClusterScopeKey(cluster1), sac.NamespaceScopeKey(namespace1)}}
		}
		if sKey == id2 {
			return [][]sac.ScopeKey{{sac.ClusterScopeKey(cluster2), sac.NamespaceScopeKey(namespace2)}}
		}
		if sKey == id3 {
			return [][]sac.ScopeKey{
				{
					sac.ClusterScopeKey(cluster1),
					sac.NamespaceScopeKey(namespace1),
				},
				{
					sac.ClusterScopeKey(cluster2),
					sac.NamespaceScopeKey(namespace2),
				},
			}
		}
		if sKey == id4 {
			return [][]sac.ScopeKey{
				{
					sac.ClusterScopeKey(cluster2),
					sac.NamespaceScopeKey(namespace3),
				},
			}
		}
		s.Fail("unexpected id")
		return nil
	}
}
