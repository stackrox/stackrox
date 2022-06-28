//go:build sql_integration
// +build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v4/pgxpool"
	clusterPostgres "github.com/stackrox/rox/central/cluster/store/cluster/postgres"
	clusterHealthPostgres "github.com/stackrox/rox/central/cluster/store/clusterhealth/postgres"
	namespace "github.com/stackrox/rox/central/namespace/datastore"
	nsPostgres "github.com/stackrox/rox/central/namespace/store/postgres"
	netEntitiesMocks "github.com/stackrox/rox/central/networkgraph/entity/datastore/mocks"
	netFlowsMocks "github.com/stackrox/rox/central/networkgraph/flow/datastore/mocks"
	nodeMocks "github.com/stackrox/rox/central/node/globaldatastore/mocks"
	"github.com/stackrox/rox/central/ranking"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
)

func TestClusterDataStoreWithPostgres(t *testing.T) {
	suite.Run(t, new(ClusterPostgresDataStoreTestSuite))
}

type ClusterPostgresDataStoreTestSuite struct {
	suite.Suite

	mockCtrl         *gomock.Controller
	ctx              context.Context
	db               *pgxpool.Pool
	nsDatastore      namespace.DataStore
	clusterDatastore DataStore
	nodeDataStore    *nodeMocks.MockGlobalDataStore
	netEntities      *netEntitiesMocks.MockEntityDataStore
	netFlows         *netFlowsMocks.MockClusterDataStore
	envIsolator      *envisolator.EnvIsolator
}

func (s *ClusterPostgresDataStoreTestSuite) SetupSuite() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")

	if !features.PostgresDatastore.Enabled() {
		s.T().Skip("Skip postgres store tests")
		s.T().SkipNow()
	}

	s.ctx = context.Background()

	source := pgtest.GetConnectionString(s.T())
	config, err := pgxpool.ParseConfig(source)
	s.Require().NoError(err)

	pool, err := pgxpool.ConnectConfig(s.ctx, config)
	s.NoError(err)
	s.db = pool

	nsPostgres.Destroy(s.ctx, s.db)
	clusterPostgres.Destroy(s.ctx, s.db)

	gormDB := pgtest.OpenGormDB(s.T(), source)
	defer pgtest.CloseGormDB(s.T(), gormDB)
	ds, err := namespace.New(nsPostgres.CreateTableAndNewStore(s.ctx, s.db, gormDB), nil, nsPostgres.NewIndexer(s.db), nil, ranking.NamespaceRanker(), nil)
	s.NoError(err)
	s.nsDatastore = ds

	s.mockCtrl = gomock.NewController(s.T())
	s.netEntities = netEntitiesMocks.NewMockEntityDataStore(s.mockCtrl)
	s.nodeDataStore = nodeMocks.NewMockGlobalDataStore(s.mockCtrl)
	s.netFlows = netFlowsMocks.NewMockClusterDataStore(s.mockCtrl)

	s.nodeDataStore.EXPECT().GetAllClusterNodeStores(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	s.netEntities.EXPECT().RegisterCluster(gomock.Any(), gomock.Any()).AnyTimes()
	clusterDS, err := New(clusterPostgres.CreateTableAndNewStore(s.ctx, s.db, gormDB), clusterHealthPostgres.CreateTableAndNewStore(s.ctx, s.db, gormDB), clusterPostgres.NewIndexer(s.db), nil, ds, nil, s.nodeDataStore, nil, nil, s.netFlows, s.netEntities, nil, nil, nil, nil, nil, nil, ranking.ClusterRanker(), nil)
	s.NoError(err)
	s.clusterDatastore = clusterDS
}

func (s *ClusterPostgresDataStoreTestSuite) TearDownSuite() {
	s.db.Close()
	s.mockCtrl.Finish()
	s.envIsolator.RestoreAll()
}

func (s *ClusterPostgresDataStoreTestSuite) TestSearchClusterStatus() {
	ctx := sac.WithAllAccess(context.Background())

	// At some point in the postgres migration, the following query did trigger an error
	// because of a missing options map in the cluster health status schema.
	// This test is there to ensure the search does not end in error for technical reasons.
	query := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ClusterStatus, storage.ClusterHealthStatus_UNHEALTHY.String()).ProtoQuery()
	res, err := s.clusterDatastore.Search(ctx, query)
	s.NoError(err)
	s.Equal(0, len(res))
}

func (s *ClusterPostgresDataStoreTestSuite) TestSearchWithPostgres() {
	ctx := sac.WithAllAccess(context.Background())

	// Upsert cluster.
	s.netFlows.EXPECT().CreateFlowStore(gomock.Any(), gomock.Any()).Return(netFlowsMocks.NewMockFlowDataStore(s.mockCtrl), nil)
	c1ID, err := s.clusterDatastore.AddCluster(ctx, &storage.Cluster{
		Name:               "c1",
		Labels:             map[string]string{"env": "prod", "team": "team"},
		MainImage:          mainImage,
		CentralApiEndpoint: centralEndpoint,
	})
	s.NoError(err)

	// Basic unscoped search.
	results, err := s.clusterDatastore.Search(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)

	// Upsert cluster.
	s.netFlows.EXPECT().CreateFlowStore(gomock.Any(), gomock.Any()).Return(netFlowsMocks.NewMockFlowDataStore(s.mockCtrl), nil)
	c2ID, err := s.clusterDatastore.AddCluster(ctx, &storage.Cluster{
		Name:               "c2",
		Labels:             map[string]string{"env": "test", "team": "team"},
		MainImage:          mainImage,
		CentralApiEndpoint: centralEndpoint,
	})
	s.NoError(err)

	// Basic unscoped search.
	results, err = s.clusterDatastore.Search(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Len(results, 2)

	ns1C1 := fixtures.GetNamespace(c1ID, "c1", "n1")
	ns2C1 := fixtures.GetNamespace(c1ID, "c1", "n2")
	ns1C2 := fixtures.GetNamespace(c2ID, "c2", "n1")

	// Upsert namespace.
	s.NoError(s.nsDatastore.AddNamespace(ctx, ns1C1))

	// Basic unscoped search.
	results, err = s.nsDatastore.Search(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)

	// Upsert.
	s.NoError(s.nsDatastore.UpdateNamespace(ctx, ns2C1))
	s.NoError(s.nsDatastore.UpdateNamespace(ctx, ns1C2))

	// Basic unscoped search.
	results, err = s.nsDatastore.Search(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Len(results, 3)

	// Query cluster with namespace search field.
	results, err = s.clusterDatastore.Search(ctx, pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n2").ProtoQuery())
	s.NoError(err)
	s.Len(results, 1)
	s.Equal(c1ID, results[0].ID)

	// Query namespace with cluster search field.
	results, err = s.nsDatastore.Search(ctx, pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "test").ProtoQuery())
	s.NoError(err)
	s.Len(results, 1)
	s.Equal(ns1C2.Id, results[0].ID)

	// Query cluster with cluster+namespace search fields.
	results, err = s.clusterDatastore.Search(ctx, pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n1").AddMapQuery(pkgSearch.ClusterLabel, "team", "team").ProtoQuery())
	s.NoError(err)
	s.Len(results, 2)
	s.ElementsMatch([]string{c1ID, c2ID}, pkgSearch.ResultsToIDs(results))

	// Query namespace with cluster+namespace search fields.
	results, err = s.nsDatastore.Search(ctx, pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n1").AddMapQuery(pkgSearch.ClusterLabel, "team", "team").ProtoQuery())
	s.NoError(err)
	s.Len(results, 2)
	s.ElementsMatch([]string{ns1C1.Id, ns1C2.Id}, pkgSearch.ResultsToIDs(results))

	// Query namespace with cluster+namespace search fields.
	results, err = s.nsDatastore.Search(ctx, pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n1").AddMapQuery(pkgSearch.ClusterLabel, "team", "blah").ProtoQuery())
	s.NoError(err)
	s.Len(results, 0)
}
