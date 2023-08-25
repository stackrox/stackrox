//go:build sql_integration

package resolvers

import (
	"context"
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestGraphQLNodeComponentEndpoints(t *testing.T) {
	suite.Run(t, new(GraphQLNodeComponentTestSuite))
}

type GraphQLNodeComponentTestSuite struct {
	suite.Suite

	ctx      context.Context
	testDB   *pgtest.TestPostgres
	resolver *Resolver
}

func (s *GraphQLNodeComponentTestSuite) SetupSuite() {

	s.ctx = loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	mockCtrl := gomock.NewController(s.T())
	s.testDB = SetupTestPostgresConn(s.T())

	nodeDS := CreateTestNodeDatastore(s.T(), s.testDB, mockCtrl)
	resolver, _ := SetupTestResolver(s.T(),
		CreateTestNodeCVEDatastore(s.T(), s.testDB),
		CreateTestNodeComponentDatastore(s.T(), s.testDB, mockCtrl),
		nodeDS,
		CreateTestNodeComponentCveEdgeDatastore(s.T(), s.testDB),
		CreateTestClusterDatastore(s.T(), s.testDB, mockCtrl, nil, nil, nodeDS),
	)
	s.resolver = resolver

	// Add test data to DataStores
	testClusters, testNodes := testClustersWithNodes()
	for _, cluster := range testClusters {
		err := s.resolver.ClusterDataStore.UpdateCluster(s.ctx, cluster)
		s.NoError(err)
	}
	for _, node := range testNodes {
		err := nodeDS.UpsertNode(s.ctx, node)
		s.NoError(err)
	}
}

func (s *GraphQLNodeComponentTestSuite) TearDownSuite() {
	s.testDB.Teardown(s.T())
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

	node := getNodeResolver(ctx, s.T(), s.resolver, fixtureconsts.Node1)
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

	node = getNodeResolver(ctx, s.T(), s.resolver, fixtureconsts.Node2)
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

	node := getNodeResolver(ctx, s.T(), s.resolver, fixtureconsts.Node1)
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

	node = getNodeResolver(ctx, s.T(), s.resolver, fixtureconsts.Node2)
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

	cluster := getClusterResolver(ctx, s.T(), s.resolver, fixtureconsts.Cluster1)
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

	cluster = getClusterResolver(ctx, s.T(), s.resolver, fixtureconsts.Cluster2)
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
	node := getNodeResolver(ctx, s.T(), s.resolver, fixtureconsts.Node2)
	lastScanned, err := comp.LastScanned(ctx)
	s.NoError(err)
	expected, err := timestamp(node.data.GetScan().GetScanTime())
	s.NoError(err)
	s.Equal(expected, lastScanned)

	// Component queried with node scope
	scopedCtx := scoped.Context(ctx, scoped.Scope{
		Level: v1.SearchCategory_NODES,
		ID:    fixtureconsts.Node1,
	})
	comp = getNodeComponentResolver(scopedCtx, s.T(), s.resolver, componentID)
	node = getNodeResolver(ctx, s.T(), s.resolver, fixtureconsts.Node1)
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
	s.ElementsMatch(idList, []string{fixtureconsts.Node1, fixtureconsts.Node2})

	count, err := comp.NodeCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(nodes)), count)

	comp = getNodeComponentResolver(ctx, s.T(), s.resolver, scancomponent.ComponentID("comp4", "1.0", ""))

	nodes, err = comp.Nodes(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(1, len(nodes))
	idList = getIDList(ctx, nodes)
	s.ElementsMatch(idList, []string{fixtureconsts.Node2})

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
