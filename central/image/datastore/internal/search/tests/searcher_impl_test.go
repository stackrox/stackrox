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
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

type searcherSuite struct {
	suite.Suite

	noAccessCtx       context.Context
	ns1ReadAccessCtx  context.Context
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

func (s *searcherSuite) SetupSuite() {
	s.noAccessCtx = sac.WithGlobalAccessScopeChecker(
		context.Background(),
		sac.DenyAllAccessScopeChecker())
	s.ns1ReadAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Image),
			sac.ClusterScopeKeys("clusterA"),
			sac.NamespaceScopeKeys("ns1")))
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

	s.badgerDB, _, err = badgerhelper.NewTemp(testutils.DBFileName(s))
	s.Require().NoError(err)

	s.store = badgerStore.New(s.badgerDB, false)

	s.searcher = New(s.store, s.indexer)
}

func (s *searcherSuite) TestNoAccess() {
	img := &storage.Image{
		Id: "img1",
		ClusternsScopes: map[string]string{
			"deploy1": sac.ClusterNSScopeString("clusterA", "ns2"),
			"deploy2": sac.ClusterNSScopeString("clusterB", "ns1"),
		},
	}

	s.Require().NoError(s.store.UpsertImage(img))
	s.Require().NoError(s.indexer.AddImage(img))

	results, err := s.searcher.Search(s.noAccessCtx, search.EmptyQuery())
	s.NoError(err)
	if features.ScopedAccessControl.Enabled() {
		s.Empty(results)
	} else {
		s.Len(results, 1)
	}

	results, err = s.searcher.Search(s.ns1ReadAccessCtx, search.EmptyQuery())
	s.NoError(err)
	if features.ScopedAccessControl.Enabled() {
		s.Empty(results)
	} else {
		s.Len(results, 1)
	}

	results, err = s.searcher.Search(s.fullReadAccessCtx, search.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)
}

func (s *searcherSuite) TestHasAccess() {
	if !features.ScopedAccessControl.Enabled() {
		s.T().SkipNow()
	}

	img := &storage.Image{
		Id: "img1",
		ClusternsScopes: map[string]string{
			"deploy1": sac.ClusterNSScopeString("clusterA", "ns1"),
			"deploy2": sac.ClusterNSScopeString("clusterB", "ns2"),
		},
	}

	s.Require().NoError(s.store.UpsertImage(img))
	s.Require().NoError(s.indexer.AddImage(img))

	results, err := s.searcher.Search(s.noAccessCtx, search.EmptyQuery())
	s.NoError(err)
	if features.ScopedAccessControl.Enabled() {
		s.Empty(results)
	} else {
		s.Len(results, 1)
	}

	results, err = s.searcher.Search(s.ns1ReadAccessCtx, search.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)

	results, err = s.searcher.Search(s.fullReadAccessCtx, search.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)
}

func (s *searcherSuite) TestNoClusterNSScopes() {
	if !features.ScopedAccessControl.Enabled() {
		s.T().SkipNow()
	}

	img := &storage.Image{
		Id: "img1",
	}

	s.Require().NoError(s.store.UpsertImage(img))
	s.Require().NoError(s.indexer.AddImage(img))

	results, err := s.searcher.Search(s.noAccessCtx, search.EmptyQuery())
	s.NoError(err)
	if features.ScopedAccessControl.Enabled() {
		s.Empty(results)
	} else {
		s.Len(results, 1)
	}

	results, err = s.searcher.Search(s.ns1ReadAccessCtx, search.EmptyQuery())
	s.NoError(err)
	if features.ScopedAccessControl.Enabled() {
		s.Empty(results)
	} else {
		s.Len(results, 1)
	}

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
	deploymentDS, err := datastore.NewBadger(s.badgerDB, s.bleveIndex, nil, nil, nil, nil, mockRiskDatastore, nil, nil)
	s.Require().NoError(err)

	clusterNSScopes := make(map[string]string)
	for _, deployment := range deployments {
		clusterNSScopes[deployment.GetId()] = sac.ClusterNSScopeStringFromObject(deployment)
		s.Require().NoError(deploymentDS.UpsertDeployment(sac.WithAllAccess(context.Background()), deployment))
	}

	img := &storage.Image{
		Id:              "img1",
		ClusternsScopes: clusterNSScopes,
	}

	s.Require().NoError(s.store.UpsertImage(img))
	s.Require().NoError(s.indexer.AddImage(img))

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, "clusterA").ProtoQuery()
	results, err := s.searcher.Search(s.ns1ReadAccessCtx, q)
	s.NoError(err)
	s.Len(results, 1)

	q = search.NewQueryBuilder().AddExactMatches(search.Namespace, "ns1").ProtoQuery()
	results, err = s.searcher.Search(s.ns1ReadAccessCtx, q)
	s.NoError(err)
	s.Len(results, 1)

	q = search.NewQueryBuilder().AddExactMatches(search.ClusterID, "clusterB").ProtoQuery()
	results, err = s.searcher.Search(s.ns1ReadAccessCtx, q)
	s.NoError(err)
	s.Empty(results)

	q = search.NewQueryBuilder().AddExactMatches(search.Namespace, "ns2").ProtoQuery()
	results, err = s.searcher.Search(s.ns1ReadAccessCtx, q)
	s.NoError(err)
	s.Empty(results)

	q = search.NewQueryBuilder().AddExactMatches(search.Namespace, "ns3").ProtoQuery()
	results, err = s.searcher.Search(s.ns1ReadAccessCtx, q)
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
