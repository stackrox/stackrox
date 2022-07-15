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
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/assert"
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

	// Upsert cluster.
	s.netFlows.EXPECT().CreateFlowStore(gomock.Any(), gomock.Any()).Return(netFlowsMocks.NewMockFlowDataStore(s.mockCtrl), nil)
	c2ID, err := s.clusterDatastore.AddCluster(ctx, &storage.Cluster{
		Name:               "c2",
		Labels:             map[string]string{"env": "test", "team": "team"},
		MainImage:          mainImage,
		CentralApiEndpoint: centralEndpoint,
	})
	s.NoError(err)

	ns1C1 := fixtures.GetNamespace(c1ID, "c1", "n1")
	ns2C1 := fixtures.GetNamespace(c1ID, "c1", "n2")
	ns1C2 := fixtures.GetNamespace(c2ID, "c2", "n1")

	// Upsert namespaces.
	s.NoError(s.nsDatastore.AddNamespace(ctx, ns1C1))
	s.NoError(s.nsDatastore.UpdateNamespace(ctx, ns2C1))
	s.NoError(s.nsDatastore.UpdateNamespace(ctx, ns1C2))

	for _, tc := range []struct {
		desc         string
		ctx          context.Context
		query        *v1.Query
		orderMatters bool
		expectedIDs  []string
		queryNs      bool
	}{
		{
			desc:         "Search clusters with empty query",
			ctx:          ctx,
			query:        pkgSearch.EmptyQuery(),
			orderMatters: false,
			expectedIDs:  []string{c1ID, c2ID},
		},
		{
			desc:         "Search clusters with cluster query",
			ctx:          ctx,
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ClusterID, c1ID).ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{c1ID},
		},
		{
			desc:         "Search clusters with namespace query",
			ctx:          ctx,
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n2").ProtoQuery(),
			orderMatters: true,
			expectedIDs:  []string{c1ID},
		},
		{
			desc:         "Search clusters with cluster+namespace query",
			ctx:          ctx,
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n1").AddMapQuery(pkgSearch.ClusterLabel, "team", "team").ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{c1ID, c2ID},
		},
		{
			desc:         "Search clusters with cluster scope",
			ctx:          scoped.Context(ctx, scoped.Scope{ID: c1ID, Level: v1.SearchCategory_CLUSTERS}),
			query:        pkgSearch.EmptyQuery(),
			orderMatters: false,
			expectedIDs:  []string{c1ID},
		},
		{
			desc:         "Search clusters with cluster scope and in-scope cluster query",
			ctx:          scoped.Context(ctx, scoped.Scope{ID: c1ID, Level: v1.SearchCategory_CLUSTERS}),
			query:        pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "prod").ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{c1ID},
		},
		{
			desc:         "Search clusters with cluster scope and out-of-scope cluster query",
			ctx:          scoped.Context(ctx, scoped.Scope{ID: c1ID, Level: v1.SearchCategory_CLUSTERS}),
			query:        pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "test").ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{},
		},
		{
			desc:         "Search clusters with cluster scope and in-scope namespace query",
			ctx:          scoped.Context(ctx, scoped.Scope{ID: c1ID, Level: v1.SearchCategory_CLUSTERS}),
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n1").ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{c1ID},
		},
		{
			desc:         "Search clusters with cluster scope and out-of-scope namespace query",
			ctx:          scoped.Context(ctx, scoped.Scope{ID: c2ID, Level: v1.SearchCategory_CLUSTERS}),
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n2").ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{},
		},
		{
			desc:         "Search clusters with namespace scope",
			ctx:          scoped.Context(ctx, scoped.Scope{ID: ns1C1.Id, Level: v1.SearchCategory_NAMESPACES}),
			query:        pkgSearch.EmptyQuery(),
			orderMatters: false,
			expectedIDs:  []string{c1ID},
		},
		{
			desc:         "Search clusters with namespace scope and in-scope cluster query",
			ctx:          scoped.Context(ctx, scoped.Scope{ID: ns1C1.Id, Level: v1.SearchCategory_NAMESPACES}),
			query:        pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "prod").ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{c1ID},
		},
		{
			desc:         "Search clusters with namespace scope and out-of-scope cluster query",
			ctx:          scoped.Context(ctx, scoped.Scope{ID: ns1C1.Id, Level: v1.SearchCategory_NAMESPACES}),
			query:        pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "test").ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{},
		},

		{
			desc:         "Search namespaces with empty query",
			ctx:          ctx,
			query:        pkgSearch.EmptyQuery(),
			orderMatters: false,
			expectedIDs:  []string{ns1C1.Id, ns2C1.Id, ns1C2.Id},
			queryNs:      true,
		},
		{
			desc:         "Search namespaces with cluster query",
			ctx:          ctx,
			query:        pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "test").ProtoQuery(),
			orderMatters: true,
			expectedIDs:  []string{ns1C2.Id},
			queryNs:      true,
		},
		{
			desc:         "Search namespaces with cluster+namespace query",
			ctx:          ctx,
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n1").AddMapQuery(pkgSearch.ClusterLabel, "team", "team").ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{ns1C1.Id, ns1C2.Id},
			queryNs:      true,
		},
		{
			desc:         "Search namespaces with cluster+namespace non-matching search fields",
			ctx:          ctx,
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n1").AddMapQuery(pkgSearch.ClusterLabel, "team", "blah").ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{},
			queryNs:      true,
		},
		{
			desc:         "Search namespace with namespace scope",
			ctx:          scoped.Context(ctx, scoped.Scope{ID: ns1C1.Id, Level: v1.SearchCategory_NAMESPACES}),
			query:        pkgSearch.EmptyQuery(),
			orderMatters: false,
			expectedIDs:  []string{ns1C1.Id},
			queryNs:      true,
		},
		{
			desc:         "Search namespace with namespace scope and in-scope cluster query",
			ctx:          scoped.Context(ctx, scoped.Scope{ID: ns1C1.Id, Level: v1.SearchCategory_NAMESPACES}),
			query:        pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "prod").ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{ns1C1.Id},
			queryNs:      true,
		},
		{
			desc:         "Search namespace with namespace scope and out-of-scope cluster query",
			ctx:          scoped.Context(ctx, scoped.Scope{ID: ns1C1.Id, Level: v1.SearchCategory_NAMESPACES}),
			query:        pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "test").ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{},
			queryNs:      true,
		},
		{
			desc:         "Search namespace with namespace scope and in-scope namespace query",
			ctx:          scoped.Context(ctx, scoped.Scope{ID: ns1C1.Id, Level: v1.SearchCategory_NAMESPACES}),
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n1").ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{ns1C1.Id},
			queryNs:      true,
		},
		{
			desc:         "Search namespace with namespace scope and out-of-scope namespace query",
			ctx:          scoped.Context(ctx, scoped.Scope{ID: ns1C1.Id, Level: v1.SearchCategory_NAMESPACES}),
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n2").ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{},
			queryNs:      true,
		},
		{
			desc:         "Search namespaces with cluster scope",
			ctx:          scoped.Context(ctx, scoped.Scope{ID: c1ID, Level: v1.SearchCategory_CLUSTERS}),
			query:        pkgSearch.EmptyQuery(),
			orderMatters: false,
			expectedIDs:  []string{ns1C1.Id, ns2C1.Id},
			queryNs:      true,
		},
		{
			desc:         "Search namespaces with cluster scope and in-scope cluster query",
			ctx:          scoped.Context(ctx, scoped.Scope{ID: c1ID, Level: v1.SearchCategory_CLUSTERS}),
			query:        pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "prod").ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{ns1C1.Id, ns2C1.Id},
			queryNs:      true,
		},
		{
			desc:         "Search namespaces with cluster scope and out-of-scope cluster query",
			ctx:          scoped.Context(ctx, scoped.Scope{ID: c1ID, Level: v1.SearchCategory_CLUSTERS}),
			query:        pkgSearch.NewQueryBuilder().AddMapQuery(pkgSearch.ClusterLabel, "env", "test").ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{},
			queryNs:      true,
		},
	} {
		s.T().Run(tc.desc, func(t *testing.T) {
			var actual []pkgSearch.Result
			var err error
			if tc.queryNs {
				actual, err = s.nsDatastore.Search(tc.ctx, tc.query)
			} else {
				actual, err = s.clusterDatastore.Search(tc.ctx, tc.query)
			}
			assert.NoError(t, err)
			assert.Len(t, actual, len(tc.expectedIDs))
			actualIDs := pkgSearch.ResultsToIDs(actual)
			if tc.orderMatters {
				assert.Equal(t, tc.expectedIDs, actualIDs)
			} else {
				assert.ElementsMatch(t, tc.expectedIDs, actualIDs)
			}
		})
	}
}
