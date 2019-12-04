package tests

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/dgraph-io/badger"
	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/globalindex"
	. "github.com/stackrox/rox/central/image/datastore/internal/search"
	"github.com/stackrox/rox/central/image/datastore/internal/store"
	badgerStore "github.com/stackrox/rox/central/image/datastore/internal/store/badger"
	"github.com/stackrox/rox/central/image/index"
	riskDatastoreMocks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/features"
	filterMocks "github.com/stackrox/rox/pkg/process/filter/mocks"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

type searcherSuite struct {
	suite.Suite

	noAccessCtx       context.Context
	fullReadAccessCtx context.Context

	badgerDB   *badger.DB
	bleveIndex bleve.Index

	store    store.Store
	indexer  index.Indexer
	searcher Searcher
}

func TestSearcher(t *testing.T) {
	suite.Run(t, new(searcherSuite))
}

func (s *searcherSuite) nsReadContext(clusterID, ns string) context.Context {
	return sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Image),
			sac.ClusterScopeKeys(clusterID),
			sac.NamespaceScopeKeys(ns)))
}

func (s *searcherSuite) SetupSuite() {
	s.noAccessCtx = sac.WithGlobalAccessScopeChecker(
		context.Background(),
		sac.DenyAllAccessScopeChecker())
	s.fullReadAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Image)))
}

func (s *searcherSuite) SetupTest() {
	var err error
	s.bleveIndex, err = globalindex.MemOnlyIndex()
	s.Require().NoError(err)

	s.indexer = index.New(s.bleveIndex)

	s.badgerDB, _, err = badgerhelper.NewTemp(testutils.DBFileName(s), features.ManagedDB.Enabled())
	s.Require().NoError(err)

	s.store = badgerStore.New(s.badgerDB, false)

	s.searcher = New(s.store, s.indexer)
}

func (s *searcherSuite) TestNoAccess() {
	mockCtrl := gomock.NewController(s.T())
	defer mockCtrl.Finish()
	mockRiskDatastore := riskDatastoreMocks.NewMockDataStore(mockCtrl)
	mockRiskDatastore.EXPECT().SearchRawRisks(gomock.Any(), gomock.Any())
	mockRiskDatastore.EXPECT().GetRisk(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	mockFilter := filterMocks.NewMockFilter(mockCtrl)
	mockFilter.EXPECT().Update(gomock.Any()).AnyTimes()

	deploymentDS, err := datastore.NewBadger(s.badgerDB, s.bleveIndex, nil, nil, nil, nil, mockRiskDatastore, nil, mockFilter)
	s.Require().NoError(err)

	deployments := []*storage.Deployment{
		{
			Id:        "deploy1",
			ClusterId: "clusterA",
			Namespace: "ns2",
			Containers: []*storage.Container{
				{
					Image: &storage.ContainerImage{
						Id: "img1",
					},
				},
			},
		},
		{
			Id:        "deploy2",
			ClusterId: "clusterB",
			Namespace: "ns1",
			Containers: []*storage.Container{
				{
					Image: &storage.ContainerImage{
						Id: "img1",
					},
				},
			},
		},
	}
	for _, deployment := range deployments {
		s.Require().NoError(deploymentDS.UpsertDeployment(sac.WithAllAccess(context.Background()), deployment))
	}

	img := &storage.Image{
		Id: "img1",
	}

	s.Require().NoError(s.store.UpsertImage(img))
	s.Require().NoError(s.indexer.AddImage(img))

	results, err := s.searcher.Search(s.noAccessCtx, search.EmptyQuery())
	s.NoError(err)
	s.Empty(results)

	results, err = s.searcher.Search(s.nsReadContext("clusterA", "ns1"), search.EmptyQuery())
	s.NoError(err)
	s.Empty(results)

	results, err = s.searcher.Search(s.nsReadContext("clusterB", "ns2"), search.EmptyQuery())
	s.NoError(err)
	s.Empty(results)

	results, err = s.searcher.Search(s.nsReadContext("clusterA", "ns2"), search.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)

	results, err = s.searcher.Search(s.nsReadContext("clusterB", "ns1"), search.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)

	results, err = s.searcher.Search(s.fullReadAccessCtx, search.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)
}

func (s *searcherSuite) TestHasAccess() {
	mockCtrl := gomock.NewController(s.T())
	defer mockCtrl.Finish()
	mockRiskDatastore := riskDatastoreMocks.NewMockDataStore(mockCtrl)
	mockRiskDatastore.EXPECT().SearchRawRisks(gomock.Any(), gomock.Any())
	mockRiskDatastore.EXPECT().GetRisk(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	mockFilter := filterMocks.NewMockFilter(mockCtrl)
	mockFilter.EXPECT().Update(gomock.Any()).AnyTimes()

	deploymentDS, err := datastore.NewBadger(s.badgerDB, s.bleveIndex, nil, nil, nil, nil, mockRiskDatastore, nil, mockFilter)
	s.Require().NoError(err)

	deployments := []*storage.Deployment{
		{
			Id:        "deploy1",
			ClusterId: "clusterA",
			Namespace: "ns1",
			Containers: []*storage.Container{
				{
					Image: &storage.ContainerImage{
						Id: "img1",
					},
				},
			},
		},
		{
			Id:        "deploy2",
			ClusterId: "clusterB",
			Namespace: "ns2",
			Containers: []*storage.Container{
				{
					Image: &storage.ContainerImage{
						Id: "img1",
					},
				},
			},
		},
	}
	for _, deployment := range deployments {
		s.Require().NoError(deploymentDS.UpsertDeployment(sac.WithAllAccess(context.Background()), deployment))
	}

	img := &storage.Image{
		Id: "img1",
	}

	s.Require().NoError(s.store.UpsertImage(img))
	s.Require().NoError(s.indexer.AddImage(img))

	results, err := s.searcher.Search(s.noAccessCtx, search.EmptyQuery())
	s.NoError(err)
	s.Empty(results)

	results, err = s.searcher.Search(s.nsReadContext("clusterA", "ns1"), search.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)

	results, err = s.searcher.Search(s.fullReadAccessCtx, search.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)
}

func (s *searcherSuite) TestPagination() {
	mockCtrl := gomock.NewController(s.T())
	defer mockCtrl.Finish()
	mockRiskDatastore := riskDatastoreMocks.NewMockDataStore(mockCtrl)
	mockRiskDatastore.EXPECT().SearchRawRisks(gomock.Any(), gomock.Any())
	mockRiskDatastore.EXPECT().GetRisk(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	mockFilter := filterMocks.NewMockFilter(mockCtrl)
	mockFilter.EXPECT().Update(gomock.Any()).AnyTimes()

	deploymentDS, err := datastore.NewBadger(s.badgerDB, s.bleveIndex, nil, nil, nil, nil, mockRiskDatastore, nil, mockFilter)
	s.Require().NoError(err)

	deployments := []*storage.Deployment{
		{
			Id:        "deploy1",
			ClusterId: "clusterA",
			Namespace: "ns1",
			Containers: []*storage.Container{
				{
					Image: &storage.ContainerImage{
						Id: "img1",
					},
				},
				{
					Image: &storage.ContainerImage{
						Id: "img2",
					},
				},
			},
		},
		{
			Id:        "deploy2",
			ClusterId: "clusterB",
			Namespace: "ns2",
			Containers: []*storage.Container{
				{
					Image: &storage.ContainerImage{
						Id: "img1",
					},
				},
				{
					Image: &storage.ContainerImage{
						Id: "img3",
					},
				},
			},
		},
	}
	for _, deployment := range deployments {
		s.Require().NoError(deploymentDS.UpsertDeployment(sac.WithAllAccess(context.Background()), deployment))
	}

	imgs := []*storage.Image{
		{Id: "img1"},
		{Id: "img2"},
		{Id: "img3"},
	}

	for _, img := range imgs {
		s.Require().NoError(s.store.UpsertImage(img))
		s.Require().NoError(s.indexer.AddImage(img))
	}

	results, err := s.searcher.Search(s.noAccessCtx, search.EmptyQuery())
	s.NoError(err)
	s.Empty(results)

	results, err = s.searcher.Search(s.nsReadContext("clusterA", "ns1"), search.EmptyQuery())
	s.NoError(err)
	s.ElementsMatch([]string{"img1", "img2"}, search.ResultsToIDs(results))

	results, err = s.searcher.Search(s.nsReadContext("clusterA", "ns2"), search.EmptyQuery())
	s.NoError(err)
	s.Empty(results)

	results, err = s.searcher.Search(s.nsReadContext("clusteB", "ns1"), search.EmptyQuery())
	s.NoError(err)
	s.Empty(results)

	results, err = s.searcher.Search(s.nsReadContext("clusterB", "ns2"), search.EmptyQuery())
	s.NoError(err)
	s.ElementsMatch([]string{"img1", "img3"}, search.ResultsToIDs(results))

	results, err = s.searcher.Search(s.fullReadAccessCtx, search.EmptyQuery())
	s.NoError(err)
	s.Len(results, 3)

	qWithPagination := search.EmptyQuery()
	qWithPagination.Pagination = &v1.QueryPagination{
		Limit: 1,
		SortOptions: []*v1.QuerySortOption{
			{Field: search.ImageSHA.String(), Reversed: true},
		},
	}

	results, err = s.searcher.Search(s.nsReadContext("clusterA", "ns1"), qWithPagination)
	s.NoError(err)
	s.Equal([]string{"img2"}, search.ResultsToIDs(results))

	results, err = s.searcher.Search(s.nsReadContext("clusterB", "ns2"), qWithPagination)
	s.NoError(err)
	s.Equal([]string{"img3"}, search.ResultsToIDs(results))

	results, err = s.searcher.Search(s.fullReadAccessCtx, qWithPagination)
	s.NoError(err)
	s.Equal([]string{"img3"}, search.ResultsToIDs(results))

	qWithPagination.Pagination.Limit = 2
	results, err = s.searcher.Search(s.nsReadContext("clusterA", "ns1"), qWithPagination)
	s.NoError(err)
	s.Equal([]string{"img2", "img1"}, search.ResultsToIDs(results))

	results, err = s.searcher.Search(s.nsReadContext("clusterB", "ns2"), qWithPagination)
	s.NoError(err)
	s.Equal([]string{"img3", "img1"}, search.ResultsToIDs(results))

	results, err = s.searcher.Search(s.fullReadAccessCtx, qWithPagination)
	s.NoError(err)
	s.Equal([]string{"img3", "img2"}, search.ResultsToIDs(results))
}

func (s *searcherSuite) TestNoClusterNSScopes() {
	img := &storage.Image{
		Id: "img1",
	}

	s.Require().NoError(s.store.UpsertImage(img))
	s.Require().NoError(s.indexer.AddImage(img))

	results, err := s.searcher.Search(s.noAccessCtx, search.EmptyQuery())
	s.NoError(err)
	s.Empty(results)

	results, err = s.searcher.Search(s.nsReadContext("clusterA", "ns1"), search.EmptyQuery())
	s.NoError(err)
	s.Empty(results)

	results, err = s.searcher.Search(s.nsReadContext("clusterA", "ns2"), search.EmptyQuery())
	s.NoError(err)
	s.Empty(results)

	results, err = s.searcher.Search(s.fullReadAccessCtx, search.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)
}

func (s *searcherSuite) TestNoSharedImageLeak() {
	// This tests that if an image is visible to a user (i.e., is used by a deployment in a namespace where the user
	// has image view access), but also used by deployments in namespaces where a user does not have image view access,
	// the image can not be found through queries that refer to fields of the latter deployments.
	deployments := []*storage.Deployment{
		{
			Id:        uuid.NewV4().String(),
			ClusterId: "clusterA",
			Namespace: "ns1",
			Containers: []*storage.Container{
				{
					Image: &storage.ContainerImage{
						Id: "img1",
					},
				},
			},
		},
		{
			Id:        uuid.NewV4().String(),
			ClusterId: "clusterA",
			Namespace: "ns2",
			Containers: []*storage.Container{
				{
					Image: &storage.ContainerImage{
						Id: "img1",
					},
				},
			},
		},
		{
			Id:        uuid.NewV4().String(),
			ClusterId: "clusterB",
			Namespace: "ns1",
			Containers: []*storage.Container{
				{
					Image: &storage.ContainerImage{
						Id: "img1",
					},
				},
			},
		},
		{
			Id:        uuid.NewV4().String(),
			ClusterId: "clusterB",
			Namespace: "ns3",
			Containers: []*storage.Container{
				{
					Image: &storage.ContainerImage{
						Id: "img1",
					},
				},
			},
		},
	}
	ctrl := gomock.NewController(s.T())
	mockRiskDatastore := riskDatastoreMocks.NewMockDataStore(ctrl)
	mockRiskDatastore.EXPECT().SearchRawRisks(gomock.Any(), gomock.Any())
	mockRiskDatastore.EXPECT().GetRisk(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	mockFilter := filterMocks.NewMockFilter(ctrl)
	mockFilter.EXPECT().Update(gomock.Any()).AnyTimes()

	deploymentDS, err := datastore.NewBadger(s.badgerDB, s.bleveIndex, nil, nil, nil, nil, mockRiskDatastore, nil, mockFilter)
	s.Require().NoError(err)

	for _, deployment := range deployments {
		s.Require().NoError(deploymentDS.UpsertDeployment(sac.WithAllAccess(context.Background()), deployment))
	}

	img := &storage.Image{
		Id: "img1",
	}

	s.Require().NoError(s.store.UpsertImage(img))
	s.Require().NoError(s.indexer.AddImage(img))

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, "clusterA").ProtoQuery()
	results, err := s.searcher.Search(s.nsReadContext("clusterA", "ns1"), q)
	s.NoError(err)
	s.Len(results, 1)

	q = search.NewQueryBuilder().AddExactMatches(search.Namespace, "ns1").ProtoQuery()
	results, err = s.searcher.Search(s.nsReadContext("clusterA", "ns1"), q)
	s.NoError(err)
	s.Len(results, 1)

	q = search.NewQueryBuilder().AddExactMatches(search.ClusterID, "clusterB").ProtoQuery()
	results, err = s.searcher.Search(s.nsReadContext("clusterA", "ns1"), q)
	s.NoError(err)
	s.Empty(results)

	q = search.NewQueryBuilder().AddExactMatches(search.Namespace, "ns2").ProtoQuery()
	results, err = s.searcher.Search(s.nsReadContext("clusterA", "ns1"), q)
	s.NoError(err)
	s.Empty(results)

	q = search.NewQueryBuilder().AddExactMatches(search.Namespace, "ns3").ProtoQuery()
	results, err = s.searcher.Search(s.nsReadContext("clusterA", "ns1"), q)
	s.NoError(err)
	s.Empty(results)

	clusterBAccessCtx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
		sac.ResourceScopeKeys(resources.Image),
		sac.ClusterScopeKeys("clusterB"),
	))
	results, err = s.searcher.Search(clusterBAccessCtx, q)
	s.NoError(err)
	s.Len(results, 1)
}
