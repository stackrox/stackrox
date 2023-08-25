//go:build sql_integration

package resolvers

import (
	"context"
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestGraphQLClusterVulnerabilityEndpoints(t *testing.T) {
	suite.Run(t, new(GraphQLClusterVulnerabilityTestSuite))
}

/*
Remaining TODO tasks:
- SubResolvers:
  - LastScanned
- Double Nested SubResolver
*/

type GraphQLClusterVulnerabilityTestSuite struct {
	suite.Suite

	ctx      context.Context
	testDB   *pgtest.TestPostgres
	resolver *Resolver

	clusterIDs []string
}

func (s *GraphQLClusterVulnerabilityTestSuite) SetupSuite() {

	s.ctx = loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	mockCtrl := gomock.NewController(s.T())
	s.testDB = SetupTestPostgresConn(s.T())

	clusterCVEDS := CreateTestClusterCVEDatastore(s.T(), s.testDB)
	nodeDatastore := CreateTestNodeDatastore(s.T(), s.testDB, mockCtrl)
	namespaceDS := CreateTestNamespaceDatastore(s.T(), s.testDB)
	resolver, _ := SetupTestResolver(s.T(),
		clusterCVEDS,
		CreateTestClusterCVEEdgeDatastore(s.T(), s.testDB),
		namespaceDS,
		CreateTestClusterDatastore(s.T(), s.testDB, mockCtrl, clusterCVEDS, namespaceDS, nodeDatastore),
	)
	s.resolver = resolver

	// Add Test Data to DataStores
	clusters := testCluster()
	s.clusterIDs = make([]string, 0, len(clusters))
	for _, c := range clusters {
		clusterID, err := s.resolver.ClusterDataStore.AddCluster(s.ctx, c)
		s.NoError(err)
		s.clusterIDs = append(s.clusterIDs, clusterID)
	}

	clusterCVEParts := testClusterCVEParts(s.clusterIDs)
	err := s.resolver.ClusterCVEDataStore.UpsertClusterCVEsInternal(s.ctx, clusterCVEParts[0].CVE.Type, clusterCVEParts...)
	s.NoError(err)
}

func (s *GraphQLClusterVulnerabilityTestSuite) TearDownSuite() {
	s.testDB.Teardown(s.T())
}

func (s *GraphQLClusterVulnerabilityTestSuite) TestUnauthorizedClusterVulnerabilityEndpoint() {
	_, err := s.resolver.ClusterVulnerability(s.ctx, IDQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLClusterVulnerabilityTestSuite) TestUnauthorizedClusterVulnerabilitiesEndpoint() {
	_, err := s.resolver.ClusterVulnerabilities(s.ctx, PaginatedQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLClusterVulnerabilityTestSuite) TestUnauthorizedClusterVulnerabilityCountEndpoint() {
	_, err := s.resolver.ClusterVulnerabilityCount(s.ctx, RawQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLClusterVulnerabilityTestSuite) TestUnauthorizedClusterVulnerabilityCounterEndpoint() {
	_, err := s.resolver.ClusterVulnerabilityCounter(s.ctx, RawQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLClusterVulnerabilityTestSuite) TestClusterVulnerabilities() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	expectedIDs := []string{"clusterCve1", "clusterCve2", "clusterCve3", "clusterCve4", "clusterCve5"}
	expectedCount := int32(len(expectedIDs))

	vulns, err := s.resolver.ClusterVulnerabilities(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(expectedCount, int32(len(vulns)))
	s.ElementsMatch(expectedIDs, getIDList(ctx, vulns))

	count, err := s.resolver.ClusterVulnerabilityCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(expectedCount, count)

	counter, err := s.resolver.ClusterVulnerabilityCounter(ctx, RawQuery{})
	s.NoError(err)
	checkVulnerabilityCounter(s.T(), counter, expectedCount, 3, 1, 2, 1, 1)
}

func (s *GraphQLClusterVulnerabilityTestSuite) TestClusterVulnerabilitiesScoped() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	type counterValues struct {
		fixable   int32
		critical  int32
		important int32
		moderate  int32
		low       int32
	}

	clusterVulnTests := []struct {
		name                  string
		id                    string
		expectedIDs           []string
		expectedCounterValues counterValues
	}{
		{
			"cluster1",
			s.clusterIDs[0],
			[]string{"clusterCve1", "clusterCve2", "clusterCve4", "clusterCve5"},
			counterValues{
				1, 1, 2, 0, 1,
			},
		},
		{
			"cluster2",
			s.clusterIDs[1],
			[]string{"clusterCve2", "clusterCve3", "clusterCve4"},
			counterValues{
				2, 1, 1, 1, 0,
			},
		},
	}

	for _, test := range clusterVulnTests {
		s.T().Run(test.name, func(t *testing.T) {
			cluster := s.getClusterResolver(ctx, test.id)
			expectedCount := int32(len(test.expectedIDs))

			vulns, err := cluster.ClusterVulnerabilities(ctx, PaginatedQuery{})
			s.NoError(err)
			s.Equal(expectedCount, int32(len(vulns)))
			s.ElementsMatch(test.expectedIDs, getIDList(ctx, vulns))

			count, err := cluster.ClusterVulnerabilityCount(ctx, RawQuery{})
			s.NoError(err)
			s.Equal(expectedCount, count)

			counter, err := cluster.ClusterVulnerabilityCounter(ctx, RawQuery{})
			s.NoError(err)
			s.Equal(test.expectedCounterValues.fixable, counter.All(ctx).Fixable(ctx))
			s.Equal(test.expectedCounterValues.critical, counter.Critical(ctx).Total(ctx))
			s.Equal(test.expectedCounterValues.important, counter.Important(ctx).Total(ctx))
			s.Equal(test.expectedCounterValues.moderate, counter.Moderate(ctx).Total(ctx))
			s.Equal(test.expectedCounterValues.low, counter.Low(ctx).Total(ctx))
		})
	}
}

func (s *GraphQLClusterVulnerabilityTestSuite) TestClusterVulnerabilitiesFixable() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	expectedIDs := []string{"clusterCve1", "clusterCve3", "clusterCve4"}
	expectedCount := int32(len(expectedIDs))

	query, err := getFixableRawQuery(true)
	s.NoError(err)

	vulns, err := s.resolver.ClusterVulnerabilities(ctx, PaginatedQuery{Query: &query})
	s.NoError(err)
	s.Equal(expectedCount, int32(len(vulns)))
	s.ElementsMatch(expectedIDs, getIDList(ctx, vulns))
	for _, vuln := range vulns {
		fixable, err := vuln.IsFixable(ctx, RawQuery{})
		s.NoError(err)
		s.Equal(true, fixable)
		// test fixed by is empty string because it requires cluster scoping
		fixedBy, err := vuln.FixedByVersion(ctx)
		s.NoError(err)
		s.Equal("", fixedBy)
	}

	count, err := s.resolver.ClusterVulnerabilityCount(ctx, RawQuery{Query: &query})
	s.NoError(err)
	s.Equal(expectedCount, count)
}

func (s *GraphQLClusterVulnerabilityTestSuite) TestClusterVulnerabilitiesNonFixable() {

	// This test fails because `clusterCve4` is fixable in one cluster and non-fixable in another
	// but gets returned in a non-fixable query (ROX-12404)
	s.T().Skip("Skipping test as a known failure (ROX-12404)")

	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	expectedIDs := []string{"clusterCve2", "clusterCve5"}
	expectedCount := int32(len(expectedIDs))

	query, err := getFixableRawQuery(false)
	s.NoError(err)

	vulns, err := s.resolver.ClusterVulnerabilities(ctx, PaginatedQuery{Query: &query})
	s.NoError(err)
	s.Equal(expectedCount, int32(len(vulns)))
	s.ElementsMatch(expectedIDs, getIDList(ctx, vulns))
	for _, vuln := range vulns {
		fixable, err := vuln.IsFixable(ctx, RawQuery{})
		s.NoError(err)
		s.Equal(false, fixable)
	}

	count, err := s.resolver.ClusterVulnerabilityCount(ctx, RawQuery{Query: &query})
	s.NoError(err)
	s.Equal(expectedCount, count)
}

func (s *GraphQLClusterVulnerabilityTestSuite) TestClusterVulnerabilitiesFixableScoped() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	clusterVulnTests := []struct {
		name        string
		id          string
		fixable     bool
		expectedIDs []string
		fixedBy     map[string]string
	}{
		{
			"cluster1fixable",
			s.clusterIDs[0],
			true,
			[]string{"clusterCve1"},
			map[string]string{"clusterCve1": "1.1"},
		},
		{
			"cluster2fixable",
			s.clusterIDs[1],
			true,
			[]string{"clusterCve3", "clusterCve4"},
			map[string]string{"clusterCve3": "1.2", "clusterCve4": "1.4"},
		},
		{
			"cluster1nonfixable",
			s.clusterIDs[0],
			false,
			[]string{"clusterCve2", "clusterCve4", "clusterCve5"},
			nil,
		},
		{
			"cluster2nonfixable",
			s.clusterIDs[1],
			false,
			[]string{"clusterCve2"},
			nil,
		},
	}

	for _, test := range clusterVulnTests {
		s.T().Run(test.name, func(t *testing.T) {
			cluster := s.getClusterResolver(ctx, test.id)
			expectedCount := int32(len(test.expectedIDs))
			query, err := getFixableRawQuery(test.fixable)
			s.NoError(err)

			vulns, err := cluster.ClusterVulnerabilities(ctx, PaginatedQuery{Query: &query})
			s.NoError(err)
			s.Equal(expectedCount, int32(len(vulns)))
			s.ElementsMatch(test.expectedIDs, getIDList(ctx, vulns))
			for _, vuln := range vulns {
				fixable, err := vuln.IsFixable(ctx, RawQuery{})
				s.NoError(err)
				s.Equal(test.fixable, fixable)

				if fixable {
					id := string(vuln.Id(ctx))
					fixedBy, err := vuln.FixedByVersion(ctx)
					s.NoError(err)
					s.Equal(test.fixedBy[id], fixedBy)
				}
			}

			count, err := cluster.ClusterVulnerabilityCount(ctx, RawQuery{Query: &query})
			s.NoError(err)
			s.Equal(expectedCount, count)
		})
	}
}

func (s *GraphQLClusterVulnerabilityTestSuite) TestClusterVulnerabilityMiss() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vulnID := graphql.ID("invalid")

	_, err := s.resolver.ClusterVulnerability(ctx, IDQuery{ID: &vulnID})
	s.Error(err)
}

func (s *GraphQLClusterVulnerabilityTestSuite) TestClusterVulnerabilityHit() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vulnID := graphql.ID("clusterCve2")

	vuln, err := s.resolver.ClusterVulnerability(ctx, IDQuery{ID: &vulnID})
	s.NoError(err)
	s.Equal(vulnID, vuln.Id(ctx))
}

func (s *GraphQLClusterVulnerabilityTestSuite) TestClusterVulnerabilityClusters() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	clusterVulnTests := []struct {
		name        string
		id          string
		expectedIDs []string
	}{
		{
			"clusterCve1",
			"clusterCve1",
			[]string{s.clusterIDs[0]},
		},
		{
			"clusterCve2",
			"clusterCve2",
			[]string{s.clusterIDs[0], s.clusterIDs[1]},
		},
		{
			"clusterCve3",
			"clusterCve3",
			[]string{s.clusterIDs[1]},
		},
	}

	for _, test := range clusterVulnTests {
		s.T().Run(test.name, func(t *testing.T) {
			expectedCount := int32(len(test.expectedIDs))

			vuln := s.getClusterVulnerabilityResolver(ctx, test.id)

			clusters, err := vuln.Clusters(ctx, PaginatedQuery{})
			s.NoError(err)
			s.Equal(expectedCount, int32(len(clusters)))
			s.ElementsMatch(test.expectedIDs, getIDList(ctx, clusters))

			count, err := vuln.ClusterCount(ctx, RawQuery{})
			s.NoError(err)
			s.Equal(expectedCount, count)
		})
	}
}

func (s *GraphQLClusterVulnerabilityTestSuite) TestVulnerabilityType() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vuln := s.getClusterVulnerabilityResolver(ctx, "clusterCve2")

	expectedTypes := []string{storage.CVE_CVEType_name[int32(storage.CVE_K8S_CVE)]}
	expectedType := storage.CVE_CVEType_name[int32(storage.CVE_K8S_CVE)]

	s.ElementsMatch(expectedTypes, vuln.VulnerabilityTypes())
	s.Equal(expectedType, vuln.VulnerabilityType())
}

func (s *GraphQLClusterVulnerabilityTestSuite) TestVectors() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vuln := s.getClusterVulnerabilityResolver(ctx, "clusterCve1")

	_, ok := vuln.Vectors().ToCVSSV2()
	s.True(ok)
	_, ok = vuln.Vectors().ToCVSSV3()
	s.False(ok)

	vuln = s.getClusterVulnerabilityResolver(ctx, "clusterCve2")

	_, ok = vuln.Vectors().ToCVSSV2()
	s.False(ok)
	_, ok = vuln.Vectors().ToCVSSV3()
	s.True(ok)

	vuln = s.getClusterVulnerabilityResolver(ctx, "clusterCve3")

	_, ok = vuln.Vectors().ToCVSSV2()
	s.False(ok)
	_, ok = vuln.Vectors().ToCVSSV3()
	s.True(ok)

	vuln = s.getClusterVulnerabilityResolver(ctx, "clusterCve4")

	s.Nil(vuln.Vectors())
}

func (s *GraphQLClusterVulnerabilityTestSuite) TestEnvImpact() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vuln := s.getClusterVulnerabilityResolver(ctx, "clusterCve1")

	impact, err := vuln.EnvImpact(ctx)
	s.NoError(err)
	s.Equal(float64(1)/8, impact)

	vuln = s.getClusterVulnerabilityResolver(ctx, "clusterCve2")

	impact, err = vuln.EnvImpact(ctx)
	s.NoError(err)
	s.Equal(float64(2)/8, impact)
}

func (s *GraphQLClusterVulnerabilityTestSuite) getClusterResolver(ctx context.Context, id string) *clusterResolver {
	clusterID := graphql.ID(id)

	cluster, err := s.resolver.Cluster(ctx, struct{ graphql.ID }{clusterID})
	s.NoError(err)
	s.Equal(clusterID, cluster.Id(ctx))
	return cluster
}

func (s *GraphQLClusterVulnerabilityTestSuite) getClusterVulnerabilityResolver(ctx context.Context, id string) ClusterVulnerabilityResolver {
	vulnID := graphql.ID(id)

	vuln, err := s.resolver.ClusterVulnerability(ctx, IDQuery{ID: &vulnID})
	s.NoError(err)
	s.Equal(vulnID, vuln.Id(ctx))
	return vuln
}
