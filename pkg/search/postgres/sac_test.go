package postgres

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
)

var (
	expectedMatchFieldQuery = v1.Query_builder{
		BaseQuery: v1.BaseQuery_builder{
			MatchFieldQuery: v1.MatchFieldQuery_builder{
				Field: "Cluster ID",
				Value: "\"clusterID\"",
			}.Build(),
		}.Build(),
	}.Build()

	expectedMatchNoneQuery = v1.Query_builder{
		BaseQuery: v1.BaseQuery_builder{
			MatchNoneQuery: &v1.MatchNoneQuery{},
		}.Build(),
	}.Build()
)

func TestGetReadWriteSACQuery(t *testing.T) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), createTestReadMultipleResourcesSomeWithNamespaceScope(t))
	got, err := GetReadWriteSACQuery(ctx, metadata("Cluster", permissions.ClusterScope))
	protoassert.Equal(t, expectedMatchFieldQuery, got)
	assert.NoError(t, err)

	got, err = GetReadWriteSACQuery(ctx, metadata("Namespace", permissions.NamespaceScope))
	protoassert.Equal(t, expectedMatchNoneQuery, got)
	assert.NoError(t, err)

	got, err = GetReadSACQuery(sac.WithNoAccess(context.Background()), metadata("Integration", permissions.GlobalScope))
	protoassert.Equal(t, expectedMatchNoneQuery, got)
	assert.NoError(t, err)

	got, err = GetReadWriteSACQuery(sac.WithNoAccess(context.Background()), metadata("Integration", permissions.GlobalScope))
	assert.Nil(t, got)
	assert.ErrorIs(t, err, sac.ErrResourceAccessDenied)
}

func metadata(name permissions.Resource, scope permissions.ResourceScope) permissions.ResourceMetadata {
	md := permissions.ResourceMetadata{
		Resource: name,
		Scope:    scope,
	}
	return md
}

func createTestReadMultipleResourcesSomeWithNamespaceScope(t *testing.T) sac.ScopeCheckerCore {
	resourceCluster := permissions.Resource("Cluster")
	resourceDeployment := permissions.Resource("Deployment")
	resourceNode := permissions.Resource("Node")

	clusterClusterID := "clusterID"
	nsNamespace2 := "namespace2"

	testScope := map[storage.Access]map[permissions.Resource]*sac.TestResourceScope{
		storage.Access_READ_WRITE_ACCESS: {
			resourceCluster: &sac.TestResourceScope{
				Included: false,
				Clusters: map[string]*sac.TestClusterScope{
					clusterClusterID: {
						Included: true,
					},
				},
			},
			resourceNode: &sac.TestResourceScope{Included: true},
			resourceDeployment: &sac.TestResourceScope{
				Included: false,
				Clusters: map[string]*sac.TestClusterScope{
					clusterClusterID: {
						Included:   false,
						Namespaces: []string{nsNamespace2},
					},
				},
			},
		},
	}
	return sac.TestScopeCheckerCoreFromFullScopeMap(t, testScope)
}
