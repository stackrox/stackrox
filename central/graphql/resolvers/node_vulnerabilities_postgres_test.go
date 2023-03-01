package resolvers

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/graph-gophers/graphql-go"
	clusterPostgres "github.com/stackrox/rox/central/cluster/store/cluster/postgres"
	nodeCVEPostgres "github.com/stackrox/rox/central/cve/node/datastore/store/postgres"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	nodeDS "github.com/stackrox/rox/central/node/datastore"
	nodePostgres "github.com/stackrox/rox/central/node/datastore/store/postgres"
	nodeComponentPostgres "github.com/stackrox/rox/central/nodecomponent/datastore/store/postgres"
	nodeComponentCVEEdgePostgres "github.com/stackrox/rox/central/nodecomponentcveedge/datastore/store/postgres"
	k8sRoleBindingDataStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	postgresSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search/scoped"
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

	ctx           context.Context
	db            *postgres.DB
	gormDB        *gorm.DB
	resolver      *Resolver
	nodeDatastore nodeDS.DataStore
}

func (s *GraphQLNodeVulnerabilityTestSuite) SetupSuite() {
	s.T().Setenv(env.PostgresDatastoreEnabled.EnvVar(), "true")

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		s.T().Skip("Skip postgres store tests")
		s.T().SkipNow()
	}

	s.ctx = loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	mockCtrl := gomock.NewController(s.T())
	s.db, s.gormDB = SetupTestPostgresConn(s.T())

	s.nodeDatastore = CreateTestNodeDatastore(s.T(), s.db, s.gormDB, mockCtrl)

	s.dropTable(postgresSchema.RoleBindingsTableName)
	s.dropTable(postgresSchema.RoleBindingsSubjectsTableName)
	//	postgresSchema.ApplySchemaForTable(s.ctx, s.gormDB, postgresSchema.RoleBindingsSubjectsTableName)
	postgresSchema.ApplySchemaForTable(s.ctx, s.gormDB, postgresSchema.RoleBindingsTableName)

	k8sRoleBindingDatastore, err := k8sRoleBindingDataStore.GetTestPostgresDataStore(s.T(), s.db)
	s.NoError(err)
	resolver, _ := SetupTestResolver(s.T(),
		CreateTestNodeCVEDatastore(s.T(), s.db, s.gormDB),
		CreateTestNodeComponentDatastore(s.T(), s.db, s.gormDB, mockCtrl),
		s.nodeDatastore,
		CreateTestNodeComponentCveEdgeDatastore(s.T(), s.db, s.gormDB),
		CreateTestClusterDatastore(s.T(), s.db, s.gormDB, mockCtrl, nil, nil, s.nodeDatastore),
		k8sRoleBindingDatastore,
	)
	s.resolver = resolver

	roleBindings := testK8sRoleBindings()
	for _, roleBinding := range roleBindings {
		err = k8sRoleBindingDatastore.UpsertRoleBinding(s.ctx, roleBinding)
		s.NoError(err)
	}

	req := searchRequest{
		Query:      "Cluster:c1",
		Categories: &[]string{"SUBJECTS"},
	}
	allowAllCtx := SetAuthorizerOverride(s.ctx, allow.Anonymous())
	results, err := resolver.SearchAutocomplete(allowAllCtx, req)
	s.NoError(err)
	fmt.Printf("%v\n", results)

	req = searchRequest{
		Query:      "Subject:subjectID1",
		Categories: &[]string{"SUBJECTS"},
	}
	results, err = resolver.SearchAutocomplete(allowAllCtx, req)
	s.NoError(err)
	fmt.Printf("%v\n", results)

	// Add test data to DataStores
	testClusters, testNodes := testClustersWithNodes()
	for _, cluster := range testClusters {
		err := s.resolver.ClusterDataStore.UpdateCluster(s.ctx, cluster)
		s.NoError(err)
	}
	for _, node := range testNodes {
		err := s.nodeDatastore.UpsertNode(s.ctx, node)
		s.NoError(err)
	}
}

func (s *GraphQLNodeVulnerabilityTestSuite) TearDownSuite() {
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
	s.Equal(int32(len(vulns)), count)

	counter, err := s.resolver.NodeVulnerabilityCounter(ctx, RawQuery{})
	s.NoError(err)
	checkVulnerabilityCounter(s.T(), counter, count, 1, 1, 2, 1, 1)
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
	vuln := getNodeVulnerabilityResolver(scopedCtx, s.T(), s.resolver, "cve-2018-1#")

	fixedBy, err := vuln.FixedByVersion(ctx)
	s.NoError(err)
	s.Equal("1.1", fixedBy)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		Level: v1.SearchCategory_NODE_COMPONENTS,
		ID:    "comp2#1.1#",
	})
	vuln = getNodeVulnerabilityResolver(scopedCtx, s.T(), s.resolver, "cve-2018-1#")

	fixedBy, err = vuln.FixedByVersion(ctx)
	s.NoError(err)
	s.Equal("1.5", fixedBy)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		Level: v1.SearchCategory_NODE_COMPONENTS,
		ID:    "comp2#1.1#",
	})
	vuln = getNodeVulnerabilityResolver(scopedCtx, s.T(), s.resolver, "cve-2017-1#")

	fixedBy, err = vuln.FixedByVersion(ctx)
	s.NoError(err)
	s.Equal("", fixedBy)
}

func (s *GraphQLNodeVulnerabilityTestSuite) TestNodeVulnerabilitiesNodeScoped() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	node := getNodeResolver(ctx, s.T(), s.resolver, fixtureconsts.Node1)
	expected := int32(3)

	vulns, err := node.NodeVulnerabilities(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(expected, int32(len(vulns)))
	idList := getIDList(ctx, vulns)
	s.ElementsMatch(idList, []string{"cve-2018-1#", "cve-2019-1#", "cve-2019-2#"})

	count, err := node.NodeVulnerabilityCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(vulns)), count)

	counter, err := node.NodeVulnerabilityCounter(ctx, RawQuery{})
	s.NoError(err)
	checkVulnerabilityCounter(s.T(), counter, count, 1, 1, 0, 1, 1)

	node = getNodeResolver(ctx, s.T(), s.resolver, fixtureconsts.Node2)
	expected = int32(5)

	vulns, err = node.NodeVulnerabilities(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(expected, int32(len(vulns)))
	idList = getIDList(ctx, vulns)
	s.ElementsMatch(idList, []string{"cve-2018-1#", "cve-2019-1#", "cve-2019-2#", "cve-2017-1#", "cve-2017-2#"})

	count, err = node.NodeVulnerabilityCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(vulns)), count)

	counter, err = node.NodeVulnerabilityCounter(ctx, RawQuery{})
	s.NoError(err)
	checkVulnerabilityCounter(s.T(), counter, count, 1, 1, 2, 1, 1)
}

func (s *GraphQLNodeVulnerabilityTestSuite) TestNodeVulnerabilitiesClusterScoped() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	cluster := getClusterResolver(ctx, s.T(), s.resolver, fixtureconsts.Cluster1)
	expected := int32(3)

	vulns, err := cluster.NodeVulnerabilities(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(expected, int32(len(vulns)))
	idList := getIDList(ctx, vulns)
	s.ElementsMatch(idList, []string{"cve-2018-1#", "cve-2019-1#", "cve-2019-2#"})

	count, err := cluster.NodeVulnerabilityCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(vulns)), count)

	counter, err := cluster.NodeVulnerabilityCounter(ctx, RawQuery{})
	s.NoError(err)
	checkVulnerabilityCounter(s.T(), counter, count, 1, 1, 0, 1, 1)

	cluster = getClusterResolver(ctx, s.T(), s.resolver, fixtureconsts.Cluster2)
	expected = int32(5)

	vulns, err = cluster.NodeVulnerabilities(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(expected, int32(len(vulns)))
	idList = getIDList(ctx, vulns)
	s.ElementsMatch(idList, []string{"cve-2018-1#", "cve-2019-1#", "cve-2019-2#", "cve-2017-1#", "cve-2017-2#"})

	count, err = cluster.NodeVulnerabilityCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(vulns)), count)

	counter, err = cluster.NodeVulnerabilityCounter(ctx, RawQuery{})
	s.NoError(err)
	checkVulnerabilityCounter(s.T(), counter, count, 1, 1, 2, 1, 1)
}

func (s *GraphQLNodeVulnerabilityTestSuite) TestPlottedNodeVulnerabilitiesNodeScoped() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	node := getNodeResolver(ctx, s.T(), s.resolver, fixtureconsts.Node1)
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

	node = getNodeResolver(ctx, s.T(), s.resolver, fixtureconsts.Node2)
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

func (s *GraphQLNodeVulnerabilityTestSuite) TestNodeVulnerabilityEnvImpact() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vuln := getNodeVulnerabilityResolver(ctx, s.T(), s.resolver, "cve-2018-1#")
	impact, err := vuln.EnvImpact(ctx)
	s.NoError(err)
	s.Equal(1.0, impact)

	vuln = getNodeVulnerabilityResolver(ctx, s.T(), s.resolver, "cve-2017-1#")
	impact, err = vuln.EnvImpact(ctx)
	s.NoError(err)
	s.Equal(0.5, impact)
}

func (s *GraphQLNodeVulnerabilityTestSuite) TestNodeVulnerabilityLastScanned() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vuln := getNodeVulnerabilityResolver(ctx, s.T(), s.resolver, "cve-2018-1#")
	node := getNodeResolver(ctx, s.T(), s.resolver, fixtureconsts.Node2)
	lastScanned, err := vuln.LastScanned(ctx)
	s.NoError(err)
	expected, err := timestamp(node.data.GetScan().GetScanTime())
	s.NoError(err)
	s.Equal(expected, lastScanned)
}

func (s *GraphQLNodeVulnerabilityTestSuite) TestNodeVulnerabilityNodes() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vuln := getNodeVulnerabilityResolver(ctx, s.T(), s.resolver, "cve-2018-1#")

	nodes, err := vuln.Nodes(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(2, len(nodes))
	idList := getIDList(ctx, nodes)
	s.ElementsMatch(idList, []string{fixtureconsts.Node1, fixtureconsts.Node2})

	count, err := vuln.NodeCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(nodes)), count)

	vuln = getNodeVulnerabilityResolver(ctx, s.T(), s.resolver, "cve-2017-1#")

	nodes, err = vuln.Nodes(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(1, len(nodes))
	idList = getIDList(ctx, nodes)
	s.ElementsMatch(idList, []string{fixtureconsts.Node2})

	count, err = vuln.NodeCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(nodes)), count)
}

func (s *GraphQLNodeVulnerabilityTestSuite) TestNodeVulnerabilityNodeComponents() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vuln := getNodeVulnerabilityResolver(ctx, s.T(), s.resolver, "cve-2018-1#")

	comps, err := vuln.NodeComponents(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(2, len(comps))
	idList := getIDList(ctx, comps)
	s.ElementsMatch(idList, []string{"comp1#0.9#", "comp2#1.1#"})

	count, err := vuln.NodeComponentCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(comps)), count)

	vuln = getNodeVulnerabilityResolver(ctx, s.T(), s.resolver, "cve-2017-1#")

	comps, err = vuln.NodeComponents(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(1, len(comps))
	idList = getIDList(ctx, comps)
	s.ElementsMatch(idList, []string{"comp4#1.0#"})

	count, err = vuln.NodeComponentCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(len(comps)), count)
}

func (s *GraphQLNodeVulnerabilityTestSuite) TestTopNodeVulnerability() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	node := getNodeResolver(ctx, s.T(), s.resolver, fixtureconsts.Node1)

	expected := graphql.ID("cve-2019-1#")
	topVuln, err := node.TopNodeVulnerability(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(expected, topVuln.Id(ctx))

	// test no error on node without any cves
	testNode := &storage.Node{
		Id:   fixtureconsts.Node3,
		Name: "node-without-cves",
		SetCves: &storage.Node_Cves{
			Cves: 0,
		},
		Scan: &storage.NodeScan{
			Components: []*storage.EmbeddedNodeScanComponent{
				{
					Name:    "comp-without-cves",
					Version: "v",
				},
			},
		},
	}
	err = s.nodeDatastore.UpsertNode(ctx, testNode)
	s.NoError(err)

	node = getNodeResolver(ctx, s.T(), s.resolver, testNode.GetId())
	topVuln, err = node.TopNodeVulnerability(ctx, RawQuery{})
	s.NoError(err)
	s.Nil(topVuln)
}

func testK8sRoleBindings() []*storage.K8SRoleBinding {
	return []*storage.K8SRoleBinding{
		{
			Id:          fixtureconsts.RoleBinding1,
			Name:        "rb1",
			ClusterName: "c1",
			ClusterId:   fixtureconsts.Cluster1,
			ClusterRole: true,
			Subjects: []*storage.Subject{
				{
					Id:          "subjectID1",
					Name:        "subjectID1",
					Kind:        storage.SubjectKind_USER,
					ClusterId:   fixtureconsts.Cluster1,
					ClusterName: "c1",
				},
				{
					Id:          "subjectID2",
					Name:        "subjectID2",
					Kind:        storage.SubjectKind_USER,
					ClusterId:   fixtureconsts.Cluster1,
					ClusterName: "c1",
				},
			},
		},
		{
			Id:          fixtureconsts.RoleBinding2,
			Name:        "rb2",
			ClusterName: "c2",
			ClusterId:   fixtureconsts.Cluster2,
			ClusterRole: true,
			Subjects: []*storage.Subject{
				{
					Id:          "subjectID3",
					Name:        "subjectID3",
					Kind:        storage.SubjectKind_USER,
					ClusterId:   fixtureconsts.Cluster1,
					ClusterName: "c2",
				},
				{
					Id:          "subjectID4",
					Name:        "subjectID4",
					Kind:        storage.SubjectKind_USER,
					ClusterId:   fixtureconsts.Cluster1,
					ClusterName: "c2",
				},
			},
		},
	}
}

func (s *GraphQLNodeVulnerabilityTestSuite) dropTable(name string) {
	sql := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", name)
	_, err := s.db.Exec(s.ctx, sql)
	s.NoError(err)
}
