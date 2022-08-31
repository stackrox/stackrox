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
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

func TestGraphQLNodeVulnerabilityEndpoints(t *testing.T) {
	suite.Run(t, new(GraphQLNodeVulnerabilityTestSuite))
}

/*
Remaining TODO tasks:
- As sub resolver when called through a deeper nesting of queries,
-       eg : Node(Id) -> Cluster -> NodeVulnerabilities, NodeComponent(Id) -> Nodes -> NodeVulnerabilities
- sub resolver values
	- vectors
*/

type GraphQLNodeVulnerabilityTestSuite struct {
	suite.Suite

	ctx      context.Context
	db       *pgxpool.Pool
	gormDB   *gorm.DB
	resolver *Resolver

	envIsolator *envisolator.EnvIsolator
}

func (s *GraphQLNodeVulnerabilityTestSuite) SetupSuite() {
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
	}
	for _, node := range testNodes {
		err = nodePostgresDataStore.UpsertNode(s.ctx, node)
		s.NoError(err)
	}
}

func (s *GraphQLNodeVulnerabilityTestSuite) TearDownSuite() {
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

func (s *GraphQLNodeVulnerabilityTestSuite) TestUnauthorizedNodeVulnerabilityEndpoint() {
	_, err := s.resolver.NodeVulnerability(s.ctx, IDQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLNodeVulnerabilityTestSuite) TestUnauthorizedNodeVulnerabilitiesEndpoint() {
	_, err := s.resolver.NodeVulnerabilities(s.ctx, PaginatedQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLNodeVulnerabilityTestSuite) TestUnauthorizedNodeVulnerabilityCountEndpoint() {
	_, err := s.resolver.NodeVulnerabilityCount(s.ctx, RawQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLNodeVulnerabilityTestSuite) TestUnauthorizedNodeVulnerabilityCounterEndpoint() {
	_, err := s.resolver.NodeVulnerabilityCounter(s.ctx, RawQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLNodeVulnerabilityTestSuite) TestUnauthorizedTopNodeVulnerabilityEndpoint() {
	_, err := s.resolver.TopNodeVulnerability(s.ctx, RawQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLNodeVulnerabilityTestSuite) TestNodeVulnerabilities() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	expected := int32(5)

	vulns, err := s.resolver.NodeVulnerabilities(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(expected, int32(len(vulns)))
	idList := getIDList(ctx, vulns)
	s.ElementsMatch(idList, []string{"cve-2018-1#", "cve-2019-1#", "cve-2019-2#", "cve-2017-1#", "cve-2017-2#"})

	count, err := s.resolver.NodeVulnerabilityCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(expected, count)

	counter, err := s.resolver.NodeVulnerabilityCounter(ctx, RawQuery{})
	s.NoError(err)
	checkVulnerabilityCounter(s.T(), counter, expected, 1, 1, 2, 1, 1)
}

func (s *GraphQLNodeVulnerabilityTestSuite) TestNodeVulnerabilitiesFixable() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	expected := int32(1)

	query, err := getFixableRawQuery(true)
	s.NoError(err)

	vulns, err := s.resolver.NodeVulnerabilities(ctx, PaginatedQuery{Query: &query})
	s.NoError(err)
	s.Equal(expected, int32(len(vulns)))
	for _, vuln := range vulns {
		fixable, err := vuln.IsFixable(ctx, RawQuery{})
		s.NoError(err)
		s.Equal(true, fixable)
		// test fixed by is empty string because it requires node component scoping
		fixedBy, err := vuln.FixedByVersion(ctx)
		s.NoError(err)
		s.Equal("", fixedBy)
	}
	idList := getIDList(ctx, vulns)
	s.ElementsMatch(idList, []string{"cve-2018-1#"})

	count, err := s.resolver.NodeVulnerabilityCount(ctx, RawQuery{Query: &query})
	s.NoError(err)
	s.Equal(expected, count)
}

func (s *GraphQLNodeVulnerabilityTestSuite) TestNodeVulnerabilitiesNonFixable() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	expected := int32(4)

	query, err := getFixableRawQuery(false)
	s.NoError(err)

	vulns, err := s.resolver.NodeVulnerabilities(ctx, PaginatedQuery{Query: &query})
	s.NoError(err)
	s.Equal(expected, int32(len(vulns)))
	for _, vuln := range vulns {
		fixable, err := vuln.IsFixable(ctx, RawQuery{})
		s.NoError(err)
		s.Equal(false, fixable)
	}
	idList := getIDList(ctx, vulns)
	s.ElementsMatch(idList, []string{"cve-2019-1#", "cve-2019-2#", "cve-2017-1#", "cve-2017-2#"})

	count, err := s.resolver.NodeVulnerabilityCount(ctx, RawQuery{Query: &query})
	s.NoError(err)
	s.Equal(expected, count)
}

func (s *GraphQLNodeVulnerabilityTestSuite) TestNodeVulnerabilitiesFixedByVersion() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	scopedCtx := scoped.Context(ctx, scoped.Scope{
		Level: v1.SearchCategory_NODE_COMPONENTS,
		ID:    "comp1#0.9#",
	})
	vuln := getNodeVulnerabilityResolver(s.T(), s.resolver, scopedCtx, "cve-2018-1#")

	fixedBy, err := vuln.FixedByVersion(ctx)
	s.NoError(err)
	s.Equal("1.1", fixedBy)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		Level: v1.SearchCategory_NODE_COMPONENTS,
		ID:    "comp2#1.1#",
	})
	vuln = getNodeVulnerabilityResolver(s.T(), s.resolver, scopedCtx, "cve-2018-1#")

	fixedBy, err = vuln.FixedByVersion(ctx)
	s.NoError(err)
	s.Equal("1.5", fixedBy)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		Level: v1.SearchCategory_NODE_COMPONENTS,
		ID:    "comp2#1.1#",
	})
	vuln = getNodeVulnerabilityResolver(s.T(), s.resolver, scopedCtx, "cve-2017-1#")

	fixedBy, err = vuln.FixedByVersion(ctx)
	s.NoError(err)
	s.Equal("", fixedBy)
}

func (s *GraphQLNodeVulnerabilityTestSuite) TestNodeVulnerabilitiesNodeScoped() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	node := getNodeResolver(s.T(), s.resolver, ctx, "nodeID1")
	expected := int32(3)

	vulns, err := node.NodeVulnerabilities(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(expected, int32(len(vulns)))
	idList := getIDList(ctx, vulns)
	s.ElementsMatch(idList, []string{"cve-2018-1#", "cve-2019-1#", "cve-2019-2#"})

	count, err := node.NodeVulnerabilityCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(expected, count)

	counter, err := node.NodeVulnerabilityCounter(ctx, RawQuery{})
	s.NoError(err)
	checkVulnerabilityCounter(s.T(), counter, expected, 1, 1, 0, 1, 1)

	node = getNodeResolver(s.T(), s.resolver, ctx, "nodeID2")
	expected = int32(5)

	vulns, err = node.NodeVulnerabilities(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(expected, int32(len(vulns)))
	idList = getIDList(ctx, vulns)
	s.ElementsMatch(idList, []string{"cve-2018-1#", "cve-2019-1#", "cve-2019-2#", "cve-2017-1#", "cve-2017-2#"})

	count, err = node.NodeVulnerabilityCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(expected, count)

	counter, err = node.NodeVulnerabilityCounter(ctx, RawQuery{})
	s.NoError(err)
	checkVulnerabilityCounter(s.T(), counter, expected, 1, 1, 2, 1, 1)
}

func (s *GraphQLNodeVulnerabilityTestSuite) TestNodeVulnerabilitiesClusterScoped() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	cluster := getClusterResolver(s.T(), s.resolver, ctx, "clusterID1")
	expected := int32(3)

	vulns, err := cluster.NodeVulnerabilities(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(expected, int32(len(vulns)))
	idList := getIDList(ctx, vulns)
	s.ElementsMatch(idList, []string{"cve-2018-1#", "cve-2019-1#", "cve-2019-2#"})

	count, err := cluster.NodeVulnerabilityCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(expected, count)

	counter, err := cluster.NodeVulnerabilityCounter(ctx, RawQuery{})
	s.NoError(err)
	checkVulnerabilityCounter(s.T(), counter, expected, 1, 1, 0, 1, 1)

	cluster = getClusterResolver(s.T(), s.resolver, ctx, "clusterID2")
	expected = int32(5)

	vulns, err = cluster.NodeVulnerabilities(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(expected, int32(len(vulns)))
	idList = getIDList(ctx, vulns)
	s.ElementsMatch(idList, []string{"cve-2018-1#", "cve-2019-1#", "cve-2019-2#", "cve-2017-1#", "cve-2017-2#"})

	count, err = cluster.NodeVulnerabilityCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(expected, count)

	counter, err = cluster.NodeVulnerabilityCounter(ctx, RawQuery{})
	s.NoError(err)
	checkVulnerabilityCounter(s.T(), counter, expected, 1, 1, 2, 1, 1)
}

func (s *GraphQLNodeVulnerabilityTestSuite) TestPlottedNodeVulnerabilitiesNodeScoped() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	node := getNodeResolver(s.T(), s.resolver, ctx, "nodeID1")
	plottedVulnRes, err := node.PlottedNodeVulnerabilities(ctx, RawQuery{})
	s.NoError(err)

	vulns, err := plottedVulnRes.NodeVulnerabilities(ctx, PaginationWrapper{})
	s.NoError(err)
	s.Equal(3, len(vulns))
	idList := getIDList(ctx, vulns)
	s.ElementsMatch(idList, []string{"cve-2018-1#", "cve-2019-1#", "cve-2019-2#"})

	basicCounter, err := plottedVulnRes.BasicNodeVulnerabilityCounter(ctx)
	s.NoError(err)
	s.Equal(int32(3), basicCounter.All(ctx).Total(ctx))
	s.Equal(int32(1), basicCounter.All(ctx).Fixable(ctx))

	node = getNodeResolver(s.T(), s.resolver, ctx, "nodeID2")
	plottedVulnRes, err = node.PlottedNodeVulnerabilities(ctx, RawQuery{})
	s.NoError(err)

	vulns, err = plottedVulnRes.NodeVulnerabilities(ctx, PaginationWrapper{})
	s.NoError(err)
	s.Equal(5, len(vulns))
	idList = getIDList(ctx, vulns)
	s.ElementsMatch(idList, []string{"cve-2018-1#", "cve-2019-1#", "cve-2019-2#", "cve-2017-1#", "cve-2017-2#"})

	basicCounter, err = plottedVulnRes.BasicNodeVulnerabilityCounter(ctx)
	s.NoError(err)
	s.Equal(int32(5), basicCounter.All(ctx).Total(ctx))
	s.Equal(int32(1), basicCounter.All(ctx).Fixable(ctx))
}

func (s *GraphQLNodeVulnerabilityTestSuite) TestNodeVulnerabilityMiss() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vulnID := graphql.ID("invalid")

	_, err := s.resolver.NodeVulnerability(ctx, IDQuery{ID: &vulnID})
	s.Error(err)
}

func (s *GraphQLNodeVulnerabilityTestSuite) TestNodeVulnerabilityHit() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vulnID := graphql.ID("cve-2018-1#")

	vuln, err := s.resolver.NodeVulnerability(ctx, IDQuery{ID: &vulnID})
	s.NoError(err)
	s.Equal(vulnID, vuln.Id(ctx))
}

func (s *GraphQLNodeVulnerabilityTestSuite) TestTopNodeVulnerabilityUnscoped() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	_, err := s.resolver.TopNodeVulnerability(ctx, RawQuery{})
	s.Error(err)
}

func (s *GraphQLNodeVulnerabilityTestSuite) TestTopNodeVulnerability() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	node := getNodeResolver(s.T(), s.resolver, ctx, "nodeID1")

	expected := graphql.ID("cve-2019-1#")
	topVuln, err := node.TopNodeVulnerability(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(expected, topVuln.Id(ctx))
}

func (s *GraphQLNodeVulnerabilityTestSuite) TestNodeVulnerabilityEnvImpact() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vuln := getNodeVulnerabilityResolver(s.T(), s.resolver, ctx, "cve-2018-1#")
	impact, err := vuln.EnvImpact(ctx)
	s.NoError(err)
	s.Equal(1.0, impact)

	vuln = getNodeVulnerabilityResolver(s.T(), s.resolver, ctx, "cve-2017-1#")
	impact, err = vuln.EnvImpact(ctx)
	s.NoError(err)
	s.Equal(0.5, impact)
}

func (s *GraphQLNodeVulnerabilityTestSuite) TestNodeVulnerabilityLastScanned() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vuln := getNodeVulnerabilityResolver(s.T(), s.resolver, ctx, "cve-2018-1#")
	node := getNodeResolver(s.T(), s.resolver, ctx, "nodeID2")
	lastScanned, err := vuln.LastScanned(ctx)
	s.NoError(err)
	expected, err := timestamp(node.data.GetScan().GetScanTime())
	s.NoError(err)
	s.Equal(expected, lastScanned)
}

func (s *GraphQLNodeVulnerabilityTestSuite) TestNodeVulnerabilityNodes() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vuln := getNodeVulnerabilityResolver(s.T(), s.resolver, ctx, "cve-2018-1#")

	nodes, err := vuln.Nodes(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(2, len(nodes))
	idList := getIDList(ctx, nodes)
	s.ElementsMatch(idList, []string{"nodeID1", "nodeID2"})

	count, err := vuln.NodeCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(nodes)), count)

	vuln = getNodeVulnerabilityResolver(s.T(), s.resolver, ctx, "cve-2017-1#")

	nodes, err = vuln.Nodes(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(1, len(nodes))
	idList = getIDList(ctx, nodes)
	s.ElementsMatch(idList, []string{"nodeID2"})

	count, err = vuln.NodeCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(nodes)), count)
}

func (s *GraphQLNodeVulnerabilityTestSuite) TestNodeVulnerabilityNodeComponents() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vuln := getNodeVulnerabilityResolver(s.T(), s.resolver, ctx, "cve-2018-1#")

	comps, err := vuln.NodeComponents(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(2, len(comps))
	idList := getIDList(ctx, comps)
	s.ElementsMatch(idList, []string{"comp1#0.9#", "comp2#1.1#"})

	count, err := vuln.NodeComponentCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(comps)), count)

	vuln = getNodeVulnerabilityResolver(s.T(), s.resolver, ctx, "cve-2017-1#")

	comps, err = vuln.NodeComponents(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(1, len(comps))
	idList = getIDList(ctx, comps)
	s.ElementsMatch(idList, []string{"comp4#1.0#"})

	count, err = vuln.NodeComponentCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(comps)), count)
}
