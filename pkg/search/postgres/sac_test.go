package postgres

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
)

func TestGetReadWriteSACQuery(t *testing.T) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), createTestReadMultipleResourcesSomeWithNamespaceScope(t))
	got, err := GetReadWriteSACQuery(ctx, metadata("Cluster", permissions.ClusterScope))
	assert.Equal(t, `base_query:<match_field_query:<field:"Cluster ID" value:"\"clusterID\"" > > `, got.String())
	assert.NoError(t, err)
	got, err = GetReadWriteSACQuery(ctx, metadata("Namespace", permissions.NamespaceScope))
	assert.Equal(t, `base_query:<match_none_query:<> > `, got.String())
	assert.NoError(t, err)
	got, err = GetReadSACQuery(sac.WithNoAccess(context.Background()), metadata("Integration", permissions.GlobalScope))
	assert.Equal(t, `base_query:<match_none_query:<> > `, got.String())
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
