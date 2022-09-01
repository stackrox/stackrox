//go:build sql_integration
// +build sql_integration

package resolvers

import (
	"context"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/graph-gophers/graphql-go"
	"github.com/jackc/pgx/v4/pgxpool"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	clusterPostgres "github.com/stackrox/rox/central/cluster/store/cluster/postgres"
	clusterHealthPostgres "github.com/stackrox/rox/central/cluster/store/clusterhealth/postgres"
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
	"github.com/stackrox/rox/central/sensor/service/connection"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stackrox/rox/pkg/search/scoped"
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
	clusterPostgres.Destroy(s.ctx, s.db)

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

	// cluster datastore
	clusterStore := clusterPostgres.CreateTableAndNewStore(s.ctx, s.db, s.gormDB)
	clusterHealthStore := clusterHealthPostgres.CreateTableAndNewStore(s.ctx, s.db, s.gormDB)
	clusterIndexer := clusterPostgres.NewIndexer(s.db)
	connMgr := connection.ManagerSingleton()
	clusterDatastore, err := clusterDataStore.New(clusterStore, clusterHealthStore, nil, nil, nil, nil, nil, nodeGlobalDatastore, nil, nil, nil, nil, nil, nil, nil, connMgr, nil, nil, ranking.NewRanker(), clusterIndexer, nil)
	s.NoError(err)
	s.resolver.ClusterDataStore = clusterDatastore

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

	// Add test data to DataStores
	testClusters, testNodes := testClustersWithNodes()
	for _, cluster := range testClusters {
		err = clusterDatastore.UpdateCluster(s.ctx, cluster)
		s.NoError(err)
	}
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
	clusterPostgres.Destroy(s.ctx, s.db)
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
	s.Equal(int32(len(comps)), count)
}

func (s *GraphQLNodeComponentTestSuite) TestNodeComponentsNodeScoped() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	node := getNodeResolver(ctx, s.T(), s.resolver, "nodeID1")
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
	s.Equal(int32(len(comps)), count)

	node = getNodeResolver(ctx, s.T(), s.resolver, "nodeID2")
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
	s.Equal(int32(len(comps)), count)
}

func (s *GraphQLNodeComponentTestSuite) TestNodeComponentsFromNodeScan() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	node := getNodeResolver(ctx, s.T(), s.resolver, "nodeID1")
	nodeScan, err := node.Scan(ctx)
	s.NoError(err)

	expected := int32(3)
	comps, err := nodeScan.NodeComponents(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(expected, int32(len(comps)))
	idList := getIDList(ctx, comps)
	s.ElementsMatch(idList, []string{
		scancomponent.ComponentID("comp1", "0.9", ""),
		scancomponent.ComponentID("comp2", "1.1", ""),
		scancomponent.ComponentID("comp3", "1.0", ""),
	})

	count, err := nodeScan.NodeComponentCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(comps)), count)

	node = getNodeResolver(ctx, s.T(), s.resolver, "nodeID2")
	nodeScan, err = node.Scan(ctx)
	s.NoError(err)

	expected = int32(3)
	comps, err = nodeScan.NodeComponents(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(expected, int32(len(comps)))
	idList = getIDList(ctx, comps)
	s.ElementsMatch(idList, []string{
		scancomponent.ComponentID("comp1", "0.9", ""),
		scancomponent.ComponentID("comp3", "1.0", ""),
		scancomponent.ComponentID("comp4", "1.0", ""),
	})

	count, err = nodeScan.NodeComponentCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(comps)), count)
}

func (s *GraphQLNodeComponentTestSuite) TestNodeComponentsClusterScoped() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	cluster := getClusterResolver(ctx, s.T(), s.resolver, "clusterID1")
	expected := int32(3)

	comps, err := cluster.NodeComponents(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(expected, int32(len(comps)))
	idList := getIDList(ctx, comps)
	s.ElementsMatch(idList, []string{
		scancomponent.ComponentID("comp1", "0.9", ""),
		scancomponent.ComponentID("comp2", "1.1", ""),
		scancomponent.ComponentID("comp3", "1.0", ""),
	})

	count, err := cluster.NodeComponentCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(comps)), count)

	cluster = getClusterResolver(ctx, s.T(), s.resolver, "clusterID2")
	expected = int32(3)

	comps, err = cluster.NodeComponents(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(expected, int32(len(comps)))
	idList = getIDList(ctx, comps)
	s.ElementsMatch(idList, []string{
		scancomponent.ComponentID("comp1", "0.9", ""),
		scancomponent.ComponentID("comp3", "1.0", ""),
		scancomponent.ComponentID("comp4", "1.0", ""),
	})

	count, err = cluster.NodeComponentCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(comps)), count)
}

func (s *GraphQLNodeComponentTestSuite) TestNodeComponentMiss() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	compID := graphql.ID("invalid")

	_, err := s.resolver.NodeComponent(ctx, IDQuery{ID: &compID})
	s.Error(err)
}

func (s *GraphQLNodeComponentTestSuite) TestNodeComponentHit() {
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
	comp := getNodeComponentResolver(ctx, s.T(), s.resolver, componentID)
	node := getNodeResolver(ctx, s.T(), s.resolver, "nodeID2")
	lastScanned, err := comp.LastScanned(ctx)
	s.NoError(err)
	expected, err := timestamp(node.data.GetScan().GetScanTime())
	s.NoError(err)
	s.Equal(expected, lastScanned)

	// Component queried with node scope
	scopedCtx := scoped.Context(ctx, scoped.Scope{
		Level: v1.SearchCategory_NODES,
		ID:    "nodeID1",
	})
	comp = getNodeComponentResolver(scopedCtx, s.T(), s.resolver, componentID)
	node = getNodeResolver(ctx, s.T(), s.resolver, "nodeID1")
	lastScanned, err = comp.LastScanned(ctx)
	s.NoError(err)
	expected, err = timestamp(node.data.GetScan().GetScanTime())
	s.NoError(err)
	s.Equal(expected, lastScanned)
}

func (s *GraphQLNodeComponentTestSuite) TestNodeComponentNodes() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	comp := getNodeComponentResolver(ctx, s.T(), s.resolver, scancomponent.ComponentID("comp1", "0.9", ""))

	nodes, err := comp.Nodes(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(2, len(nodes))
	idList := getIDList(ctx, nodes)
	s.ElementsMatch(idList, []string{"nodeID1", "nodeID2"})

	count, err := comp.NodeCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(nodes)), count)

	comp = getNodeComponentResolver(ctx, s.T(), s.resolver, scancomponent.ComponentID("comp4", "1.0", ""))

	nodes, err = comp.Nodes(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(1, len(nodes))
	idList = getIDList(ctx, nodes)
	s.ElementsMatch(idList, []string{"nodeID2"})

	count, err = comp.NodeCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(nodes)), count)
}

func (s *GraphQLNodeComponentTestSuite) TestNodeComponentNodeVulnerabilities() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	comp := getNodeComponentResolver(ctx, s.T(), s.resolver, scancomponent.ComponentID("comp3", "1.0", ""))
	vulns, err := comp.NodeVulnerabilities(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(2, len(vulns))
	idList := getIDList(ctx, vulns)
	s.ElementsMatch(idList, []string{"cve-2019-1#", "cve-2019-2#"})

	count, err := comp.NodeVulnerabilityCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(vulns)), count)

	counter, err := comp.NodeVulnerabilityCounter(ctx, RawQuery{})
	s.NoError(err)
	checkVulnerabilityCounter(s.T(), counter, count, 0, 0, 0, 1, 1)

	comp = getNodeComponentResolver(ctx, s.T(), s.resolver, scancomponent.ComponentID("comp1", "0.9", ""))
	vulns, err = comp.NodeVulnerabilities(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(1, len(vulns))
	idList = getIDList(ctx, vulns)
	s.ElementsMatch(idList, []string{"cve-2018-1#"})

	count, err = comp.NodeVulnerabilityCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(vulns)), count)

	counter, err = comp.NodeVulnerabilityCounter(ctx, RawQuery{})
	s.NoError(err)
	checkVulnerabilityCounter(s.T(), counter, count, 1, 1, 0, 0, 0)
}

func (s *GraphQLNodeComponentTestSuite) TestNodeComponentPlottedNodeVulnerabilities() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	comp := getNodeComponentResolver(ctx, s.T(), s.resolver, scancomponent.ComponentID("comp3", "1.0", ""))
	plottedVulnRes, err := comp.PlottedNodeVulnerabilities(ctx, RawQuery{})
	s.NoError(err)

	vulns, err := plottedVulnRes.NodeVulnerabilities(ctx, PaginationWrapper{})
	s.NoError(err)
	s.Equal(2, len(vulns))
	idList := getIDList(ctx, vulns)
	s.ElementsMatch(idList, []string{"cve-2019-1#", "cve-2019-2#"})

	basicCounter, err := plottedVulnRes.BasicNodeVulnerabilityCounter(ctx)
	s.NoError(err)
	s.Equal(int32(2), basicCounter.All(ctx).Total(ctx))
	s.Equal(int32(0), basicCounter.All(ctx).Fixable(ctx))

	comp = getNodeComponentResolver(ctx, s.T(), s.resolver, scancomponent.ComponentID("comp1", "0.9", ""))
	plottedVulnRes, err = comp.PlottedNodeVulnerabilities(ctx, RawQuery{})
	s.NoError(err)

	vulns, err = plottedVulnRes.NodeVulnerabilities(ctx, PaginationWrapper{})
	s.NoError(err)
	s.Equal(1, len(vulns))
	idList = getIDList(ctx, vulns)
	s.ElementsMatch(idList, []string{"cve-2018-1#"})

	basicCounter, err = plottedVulnRes.BasicNodeVulnerabilityCounter(ctx)
	s.NoError(err)
	s.Equal(int32(1), basicCounter.All(ctx).Total(ctx))
	s.Equal(int32(1), basicCounter.All(ctx).Fixable(ctx))
}

func (s *GraphQLNodeComponentTestSuite) NodeComponentTopNodeVulnerability() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	comp := getNodeComponentResolver(ctx, s.T(), s.resolver, scancomponent.ComponentID("comp3", "1.0", ""))

	expected := graphql.ID("cve-2019-1#")
	topVuln, err := comp.TopNodeVulnerability(ctx)
	s.NoError(err)
	s.Equal(expected, topVuln.Id(ctx))
}
