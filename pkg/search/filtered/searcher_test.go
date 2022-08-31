package filtered

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/graph/testutils"
	"github.com/stackrox/rox/pkg/dbhelper"
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

	nsHandler  = &dbhelper.BucketHandler{BucketPrefix: []byte("ns:")}
	objHandler = &dbhelper.BucketHandler{BucketPrefix: []byte("obj")}

	objToNSPath = dackbox.BackwardsBucketPath(objHandler, nsHandler)

	testGraph = testutils.GraphFromPaths(
		objToNSPath.KeyPath(id1, namespace1),
		objToNSPath.KeyPath(id2, namespace2),
		objToNSPath.KeyPath(id3, namespace1),
		objToNSPath.KeyPath(id3, namespace2),
		objToNSPath.KeyPath(id4, namespace3),
	)

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
	s.mockUnsafeSearcher.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{
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

	testutils.DoWithGraph(ctx, testGraph, func(ctx context.Context) {
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
	})
}

func (s *filteredSearcherTestSuite) TestGlobalDenied() {
	s.mockUnsafeSearcher.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{
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

	testutils.DoWithGraph(ctx, testGraph, func(ctx context.Context) {
		searcher := UnsafeSearcher(s.mockUnsafeSearcher, filter)
		results, err := searcher.Search(ctx, &v1.Query{})
		s.NoError(err, "search should have succeeded")
		s.Equal([]search.Result{}, results)
	})
}

func (s *filteredSearcherTestSuite) TestScoped() {
	s.mockUnsafeSearcher.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{
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

	testutils.DoWithGraph(ctx, testGraph, func(ctx context.Context) {
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
	})
}

func (s *filteredSearcherTestSuite) TestMultiScoped() {
	s.mockUnsafeSearcher.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{
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

	testutils.DoWithGraph(ctx, testGraph, func(ctx context.Context) {
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
	})
}

func (s *filteredSearcherTestSuite) fakeScopeFunc(_ context.Context, id string) []sac.ScopeKey {
	switch id {
	case namespace1:
		return []sac.ScopeKey{sac.ClusterScopeKey(cluster1), sac.NamespaceScopeKey(namespace1)}
	case namespace2:
		return []sac.ScopeKey{sac.ClusterScopeKey(cluster2), sac.NamespaceScopeKey(namespace2)}
	case namespace3:
		return []sac.ScopeKey{sac.ClusterScopeKey(cluster2), sac.NamespaceScopeKey(namespace3)}
	}
	s.Fail("unexpected namespace ID", id)
	return nil
}

func (s *filteredSearcherTestSuite) fakeTransformer() ScopeTransform {
	return ScopeTransform{
		Path:      objToNSPath,
		ScopeFunc: s.fakeScopeFunc,
	}
}
