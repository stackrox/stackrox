//go:build sql_integration

package resolvers

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/cluster/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
)

func TestGraphQLClustersPostgres(t *testing.T) {
	suite.Run(t, new(graphQLClusterTestSuite))
}

type graphQLClusterTestSuite struct {
	suite.Suite

	testDB   *pgtest.TestPostgres
	resolver *Resolver

	clusters []*storage.Cluster

	scopeObjects []*v1.ScopeObject
}

func (s *graphQLClusterTestSuite) SetupSuite() {
	s.testDB = pgtest.ForT(s.T())

	clusterDataStore, err := datastore.GetTestPostgresDataStore(s.T(), s.testDB)
	s.Require().NoError(err)

	s.resolver = &Resolver{ClusterDataStore: clusterDataStore}

	ctx := sac.WithAllAccess(context.Background())
	for i := 0; i < 6; i++ {
		cluster := fixtures.GetCluster(fmt.Sprintf("Test cluster %d", i+1))
		id, addErr := clusterDataStore.AddCluster(ctx, cluster)
		s.Require().NoError(addErr)
		cluster.Id = id
		s.clusters = append(s.clusters, cluster)
		s.addClusterScopeObject(cluster)
	}
}

func (s *graphQLClusterTestSuite) addClusterScopeObject(cluster *storage.Cluster) {
	scopeObject := &v1.ScopeObject{
		Id:   cluster.Id,
		Name: cluster.Name,
	}
	s.scopeObjects = append(s.scopeObjects, scopeObject)
}

func (s *graphQLClusterTestSuite) TestClustersForPermission() {
	testCases := map[string]struct {
		ctx            context.Context
		targetResource permissions.ResourceMetadata

		expectedScopeObjects []*v1.ScopeObject
	}{
		"Full Access, All cluster retrieved": {
			ctx:            sac.WithAllAccess(context.Background()),
			targetResource: resources.Compliance,

			expectedScopeObjects: s.scopeObjects,
		},
		"Unrestricted Read on target resource, All clusters retrieved": {
			ctx: sac.WithGlobalAccessScopeChecker(
				context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Compliance),
				),
			),
			targetResource:       resources.Compliance,
			expectedScopeObjects: s.scopeObjects,
		},
		"Partial Read on target resource, All allowed clusters retrieved": {
			ctx: sac.WithGlobalAccessScopeChecker(
				context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Compliance),
					sac.ClusterScopeKeys(s.clusters[1].Id, s.clusters[2].Id, s.clusters[4].Id),
				),
			),
			targetResource:       resources.Compliance,
			expectedScopeObjects: []*v1.ScopeObject{s.scopeObjects[1], s.scopeObjects[2], s.scopeObjects[4]},
		},
		"Partial Read on target resource, only allowed cluster retrieved": {
			ctx: sac.WithGlobalAccessScopeChecker(
				context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Compliance),
					sac.ClusterScopeKeys(s.clusters[3].Id),
				),
			),
			targetResource:       resources.Compliance,
			expectedScopeObjects: []*v1.ScopeObject{s.scopeObjects[3]},
		},
		"Disallowed Read on target resource, no cluster retrieved": {
			ctx: sac.WithGlobalAccessScopeChecker(
				context.Background(),
				sac.TestScopeCheckerCoreFromFullScopeMap(
					s.T(),
					sac.TestScopeMap{
						storage.Access_READ_ACCESS: map[permissions.Resource]*sac.TestResourceScope{
							resources.Compliance.Resource: {
								Included: false,
							},
						},
					},
				),
			),
			targetResource:       resources.Compliance,
			expectedScopeObjects: []*v1.ScopeObject{},
		},
		"Unrestricted Read on wrong resource, No cluster retrieved": {
			ctx: sac.WithGlobalAccessScopeChecker(
				context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.DeploymentExtension),
				),
			),
			targetResource:       resources.Compliance,
			expectedScopeObjects: []*v1.ScopeObject{},
		},
	}

	for testName, testCase := range testCases {
		s.Run(testName, func() {
			ctx := testCase.ctx
			query := PaginatedQuery{}
			targetResource := testCase.targetResource

			objectResolvers, err := s.resolver.clustersForReadPermission(ctx, query, targetResource)
			s.NoError(err)
			scopeObjects := make([]*v1.ScopeObject, 0, len(objectResolvers))
			for _, objectResolver := range objectResolvers {
				scopeObjects = append(scopeObjects, objectResolver.data)
			}
			protoassert.ElementsMatch(s.T(), testCase.expectedScopeObjects, scopeObjects)
		})
	}
}
