package resolvers

import (
	"context"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/graph-gophers/graphql-go"
	"github.com/jackc/pgx/v4/pgxpool"
	nodeCVEDataStore "github.com/stackrox/rox/central/cve/node/datastore"
	nodeCVESearch "github.com/stackrox/rox/central/cve/node/datastore/search"
	nodeCVEPostgres "github.com/stackrox/rox/central/cve/node/datastore/store/postgres"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	nodeDackboxDataStore "github.com/stackrox/rox/central/node/datastore/dackbox/datastore"
	nodeGlobalDataStore "github.com/stackrox/rox/central/node/datastore/dackbox/globaldatastore"
	nodeSearch "github.com/stackrox/rox/central/node/datastore/search"
	nodePostgres "github.com/stackrox/rox/central/node/datastore/store/postgres"
	nodeComponentDataStore "github.com/stackrox/rox/central/nodecomponent/datastore"
	nodeComponentSearch "github.com/stackrox/rox/central/nodecomponent/datastore/search"
	nodeComponentPostgres "github.com/stackrox/rox/central/nodecomponent/datastore/store/postgres"
	nodeComponentCVEEdgeDataStore "github.com/stackrox/rox/central/nodecomponentcveedge/datastore"
	nodeComponentCVEEdgeSearch "github.com/stackrox/rox/central/nodecomponentcveedge/datastore/search"
	nodeComponentCVEEdgePostgres "github.com/stackrox/rox/central/nodecomponentcveedge/datastore/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	mockRisks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

func TestGraphQLNodeComponentEndpoints(t *testing.T) {
	suite.Run(t, new(GraphQLNodeComponentTestSuite))
}

type GraphQLNodeComponentTestSuite struct {
	suite.Suite

	ctx      context.Context
	db       *pgxpool.Pool
	gormDB   *gorm.DB
	resolver *Resolver

	envIsolator *envisolator.EnvIsolator
}

func (s *GraphQLNodeComponentTestSuite) SetupSuite() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")

	if !features.PostgresDatastore.Enabled() {
		s.T().Skip("Skip postgres store tests")
		s.T().SkipNow()
	}

	s.ctx = context.Background()

	source := pgtest.GetConnectionString(s.T())
	config, err := pgxpool.ParseConfig(source)
	s.NoError(err)

	pool, err := pgxpool.ConnectConfig(s.ctx, config)
	s.NoError(err)
	s.gormDB = pgtest.OpenGormDB(s.T(), source)
	s.db = pool

	// destroy datastores if they exist
	nodePostgres.Destroy(s.ctx, s.db)
	nodeComponentPostgres.Destroy(s.ctx, s.db)
	nodeCVEPostgres.Destroy(s.ctx, s.db)
	nodeComponentCVEEdgePostgres.Destroy(s.ctx, s.db)

	// create mock resolvers, set relevant ones
	s.resolver = NewMock()

	// nodeCVE datastore
	nodeCVEStore := nodeCVEPostgres.CreateTableAndNewStore(s.ctx, s.db, s.gormDB)
	nodeCVEIndexer := nodeCVEPostgres.NewIndexer(s.db)
	nodeCVESearcher := nodeCVESearch.New(nodeCVEStore, nodeCVEIndexer)
	nodeCVEDatastore, err := nodeCVEDataStore.New(nodeCVEStore, nodeCVEIndexer, nodeCVESearcher, concurrency.NewKeyFence())
	s.NoError(err, "Failed to create nodeCVEDatastore")
	s.resolver.NodeCVEDataStore = nodeCVEDatastore

	// node datastore
	riskMock := mockRisks.NewMockDataStore(gomock.NewController(s.T()))
	nodeStore := nodePostgres.CreateTableAndNewStore(s.ctx, s.T(), s.db, s.gormDB, false)
	nodeIndexer := nodePostgres.NewIndexer(s.db)
	nodeSearcher := nodeSearch.NewV2(nodeStore, nodeIndexer)
	nodePostgresDataStore := nodeDackboxDataStore.NewWithPostgres(nodeStore, nodeIndexer, nodeSearcher, riskMock, ranking.NewRanker(), ranking.NewRanker())
	nodeGlobalDatastore, err := nodeGlobalDataStore.New(nodePostgresDataStore)
	s.NoError(err, "Failed to create nodeGlobalDatastore")
	s.resolver.NodeGlobalDataStore = nodeGlobalDatastore

	// nodeComponent datastore
	nodeCompStore := nodeComponentPostgres.CreateTableAndNewStore(s.ctx, s.db, s.gormDB)
	nodeCompIndexer := nodeComponentPostgres.NewIndexer(s.db)
	nodeCompSearcher := nodeComponentSearch.New(nodeCompStore, nodeCompIndexer)
	s.resolver.NodeComponentDataStore = nodeComponentDataStore.New(nodeCompStore, nodeCompIndexer, nodeCompSearcher, riskMock, ranking.NewRanker())

	// nodeComponentCVEEdge datastore
	nodeComponentCveEdgeStore := nodeComponentCVEEdgePostgres.CreateTableAndNewStore(s.ctx, s.db, s.gormDB)
	nodeCompontCveEdgeIndexer := nodeComponentCVEEdgePostgres.NewIndexer(s.db)
	nodeComponentCveEdgeSearcher := nodeComponentCVEEdgeSearch.New(nodeComponentCveEdgeStore, nodeCompontCveEdgeIndexer)
	nodeComponentCveEdgeDatastore, err := nodeComponentCVEEdgeDataStore.New(nodeComponentCveEdgeStore, nodeCompontCveEdgeIndexer, nodeComponentCveEdgeSearcher)
	s.NoError(err)
	s.resolver.NodeComponentCVEEdgeDataStore = nodeComponentCveEdgeDatastore

	// Sac permissions
	s.ctx = sac.WithAllAccess(s.ctx)

	// loaders used by graphql layer
	loaders.RegisterTypeFactory(reflect.TypeOf(storage.Node{}), func() interface{} {
		return loaders.NewNodeLoader(nodePostgresDataStore)
	})
	loaders.RegisterTypeFactory(reflect.TypeOf(storage.NodeComponent{}), func() interface{} {
		return loaders.NewNodeComponentLoader(s.resolver.NodeComponentDataStore)
	})
	loaders.RegisterTypeFactory(reflect.TypeOf(storage.NodeCVE{}), func() interface{} {
		return loaders.NewNodeCVELoader(s.resolver.NodeCVEDataStore)
	})
	s.ctx = loaders.WithLoaderContext(s.ctx)

	// Add Test Data to DataStores
	testNodes := testNodes()
	for _, node := range testNodes {
		err = nodePostgresDataStore.UpsertNode(s.ctx, node)
		s.NoError(err)
	}
}

func (s *GraphQLNodeComponentTestSuite) TearDownSuite() {
	s.envIsolator.RestoreAll()

	nodePostgres.Destroy(s.ctx, s.db)
	nodeComponentPostgres.Destroy(s.ctx, s.db)
	nodeCVEPostgres.Destroy(s.ctx, s.db)
	nodeComponentCVEEdgePostgres.Destroy(s.ctx, s.db)
	pgtest.CloseGormDB(s.T(), s.gormDB)
	s.db.Close()
}

// permission checks

func (s *GraphQLNodeComponentTestSuite) TestUnauthorizedNodeComponentEndpoint() {
	_, err := s.resolver.NodeComponent(s.ctx, IDQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLNodeComponentTestSuite) TestUnauthorizedNodeComponentsEndpoint() {
	_, err := s.resolver.NodeComponents(s.ctx, PaginatedQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLNodeComponentTestSuite) TestUnauthorizedNodeComponentCountEndpoint() {
	_, err := s.resolver.NodeComponentCount(s.ctx, RawQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLNodeComponentTestSuite) TestNodeComponents() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	expected := int32(4)

	comps, err := s.resolver.NodeComponents(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(expected, int32(len(comps)))
	idList := getIDList(ctx, comps)
	s.ElementsMatch(idList, []string{
		scancomponent.ComponentID("comp1", "0.9", ""),
		scancomponent.ComponentID("comp2", "1.1", ""),
		scancomponent.ComponentID("comp3", "1.0", ""),
		scancomponent.ComponentID("comp4", "1.0", ""),
	})

	count, err := s.resolver.NodeComponentCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(expected, count)
}

func (s *GraphQLNodeComponentTestSuite) TestNodeComponentsScoped() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	node := s.getNodeResolver(ctx, "id1")
	expected := int32(3)

	comps, err := node.NodeComponents(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(expected, int32(len(comps)))
	idList := getIDList(ctx, comps)
	s.ElementsMatch(idList, []string{
		scancomponent.ComponentID("comp1", "0.9", ""),
		scancomponent.ComponentID("comp2", "1.1", ""),
		scancomponent.ComponentID("comp3", "1.0", ""),
	})

	count, err := node.NodeComponentCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(expected, count)

	node = s.getNodeResolver(ctx, "id2")
	expected = int32(3)

	comps, err = node.NodeComponents(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(expected, int32(len(comps)))
	idList = getIDList(ctx, comps)
	s.ElementsMatch(idList, []string{
		scancomponent.ComponentID("comp1", "0.9", ""),
		scancomponent.ComponentID("comp3", "1.0", ""),
		scancomponent.ComponentID("comp4", "1.0", ""),
	})

	count, err = node.NodeComponentCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(expected, count)
}

func (s *GraphQLNodeComponentTestSuite) TestNodeVulnerabilityMiss() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	compID := graphql.ID("invalid")

	_, err := s.resolver.NodeComponent(ctx, IDQuery{ID: &compID})
	s.Error(err)
}

func (s *GraphQLNodeComponentTestSuite) TestNodeVulnerabilityHit() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	compID := graphql.ID(scancomponent.ComponentID("comp1", "0.9", ""))

	comp, err := s.resolver.NodeComponent(ctx, IDQuery{ID: &compID})
	s.NoError(err)
	s.Equal(compID, comp.Id(ctx))
}

func (s *GraphQLNodeComponentTestSuite) TestNodeComponentLastScanned() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	componentID := scancomponent.ComponentID("comp1", "0.9", "")

	// Component queried unscoped
	comp := s.getNodeComponentResolver(ctx, componentID)
	node := s.getNodeResolver(ctx, "id2")
	lastScanned, err := comp.LastScanned(ctx)
	s.NoError(err)
	expected, err := timestamp(node.data.GetScan().GetScanTime())
	s.NoError(err)
	s.Equal(expected, lastScanned)

	// Component queried with node scope
	node = s.getNodeResolver(ctx, "id1")
	query := search.NewQueryBuilder().AddExactMatches(search.ComponentID, componentID).Query()
	comps, err := node.NodeComponents(ctx, PaginatedQuery{Query: &query})
	s.NoError(err)
	s.Equal(1, len(comps))
	lastScanned, err = comps[0].LastScanned(ctx)
	s.NoError(err)
	expected, err = timestamp(node.data.GetScan().GetScanTime())
	s.NoError(err)
	s.Equal(expected, lastScanned)
}

func (s *GraphQLNodeComponentTestSuite) TestNodeComponentNodes() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	comp := s.getNodeComponentResolver(ctx, scancomponent.ComponentID("comp1", "0.9", ""))

	nodes, err := comp.Nodes(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(2, len(nodes))
	idList := getIDList(ctx, nodes)
	s.ElementsMatch(idList, []string{"id1", "id2"})

	count, err := comp.NodeCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(nodes)), count)

	comp = s.getNodeComponentResolver(ctx, scancomponent.ComponentID("comp4", "1.0", ""))

	nodes, err = comp.Nodes(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(1, len(nodes))
	idList = getIDList(ctx, nodes)
	s.ElementsMatch(idList, []string{"id2"})

	count, err = comp.NodeCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(nodes)), count)
}

func (s *GraphQLNodeComponentTestSuite) getNodeResolver(ctx context.Context, id string) *nodeResolver {
	nodeID := graphql.ID(id)

	node, err := s.resolver.Node(ctx, struct{ graphql.ID }{nodeID})
	s.NoError(err)
	s.Equal(nodeID, node.Id(ctx))
	return node
}

func (s *GraphQLNodeComponentTestSuite) getNodeComponentResolver(ctx context.Context, id string) NodeComponentResolver {
	compID := graphql.ID(id)

	comp, err := s.resolver.NodeComponent(ctx, IDQuery{ID: &compID})
	s.NoError(err)
	s.Equal(compID, comp.Id(ctx))
	return comp
}
