package dackbox

import (
	"context"

	clusterDackBox "github.com/stackrox/rox/central/cluster/dackbox"
	cveDackBox "github.com/stackrox/rox/central/cve/dackbox"
	nsDackBox "github.com/stackrox/rox/central/namespace/dackbox"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/dackbox/keys/transformation"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search/filtered"
)

var log = logging.LoggerForModule()

var (
	// NamespaceSACTransform transforms namespace ids into their SAC scopes.
	NamespaceSACTransform = namespaceScoped(NamespaceTransformations[v1.SearchCategory_NAMESPACES])

	// DeploymentSACTransform transforms deployment ids into their SAC scopes.
	DeploymentSACTransform = namespaceScoped(DeploymentTransformations[v1.SearchCategory_NAMESPACES])

	// ImageSACTransform transforms image ids into their SAC scopes.
	ImageSACTransform = namespaceScoped(ImageTransformations[v1.SearchCategory_NAMESPACES])

	// ImageComponentEdgeSACTransform transforms image:component edge ids into their SAC scopes.
	ImageComponentEdgeSACTransform = namespaceScoped(ImageComponentEdgeTransformations[v1.SearchCategory_NAMESPACES])

	// ComponentSACTransform transforms component ids into their SAC scopes.
	ComponentSACTransform = namespaceScoped(ComponentTransformations[v1.SearchCategory_NAMESPACES])

	// ComponentVulnEdgeSACTransform transforms component:vulnerability edge ids into their SAC scopes.
	ComponentVulnEdgeSACTransform = namespaceScoped(ComponentCVEEdgeTransformations[v1.SearchCategory_NAMESPACES])

	// VulnSACTransform transforms component vulnerability ids into their SAC scopes.
	VulnSACTransform = namespaceScoped(CVETransformations[v1.SearchCategory_NAMESPACES])

	// ClusterVulnEdgeSACTransform transforms cluster:vulnerability edge ids into their SAC scopes.
	ClusterVulnEdgeSACTransform = clusterScoped(cveToClustersWithoutDeployments)

	// ClusterVulnSACTransform transforms cluster vulnerability ids into their SAC scopes.
	ClusterVulnSACTransform = clusterScoped(cveToClustersWithoutDeployments)
)

func clusterScoped(toClusterIDs transformation.OneToMany) filtered.ScopeTransform {
	return func(ctx context.Context, keys []byte) [][]sac.ScopeKey {
		clusterIDs := toClusterIDs(ctx, keys)
		return clusterIDsToScopes(ctx, clusterIDs)
	}
}

func clusterIDsToScopes(_ context.Context, clusterIDs [][]byte) [][]sac.ScopeKey {
	ret := make([][]sac.ScopeKey, 0, len(clusterIDs))
	for _, clusterID := range clusterIDs {
		ret = append(ret, []sac.ScopeKey{sac.ClusterScopeKey(clusterID)})
	}
	return ret
}

func namespaceScoped(toNamespaceIDs transformation.OneToMany) filtered.ScopeTransform {
	return func(ctx context.Context, keys []byte) [][]sac.ScopeKey {
		namespaceIDs := toNamespaceIDs(ctx, keys)
		return namespaceIDsToScopes(ctx, namespaceIDs)
	}
}

func namespaceIDsToScopes(ctx context.Context, namespaceIDs [][]byte) [][]sac.ScopeKey {
	ret := make([][]sac.ScopeKey, 0, len(namespaceIDs))
	for _, namespaceID := range namespaceIDs {
		namespace := namespaceIDToNamespaces(ctx, namespaceID)
		if len(namespace) == 0 {
			continue
		}
		if len(namespace) > 1 {
			log.Errorf("namespace id %s had multiple namespaces", namespaceID)
			continue
		}
		cluster := GraphTransformations[v1.SearchCategory_NAMESPACES][v1.SearchCategory_CLUSTERS](ctx, namespaceID)
		if len(cluster) == 0 {
			continue
		}
		if len(cluster) > 1 {
			log.Errorf("namespace id %s had multiple clusters", namespaceID)
			continue
		}
		ret = append(ret, []sac.ScopeKey{sac.ClusterScopeKey(cluster[0]), sac.NamespaceScopeKey(namespace[0])})
	}
	return ret
}

// This transforms the namespace ID to the namespace with the graph.
var namespaceIDToNamespaces = transformation.AddPrefix(nsDackBox.Bucket).
	ThenMapToMany(transformation.ForwardFromContext()).
	Then(transformation.HasPrefix(nsDackBox.SACBucket)).
	ThenMapEachToOne(transformation.StripPrefix(nsDackBox.SACBucket))

var cveToClustersWithoutDeployments = transformation.AddPrefix(cveDackBox.Bucket).
	ThenMapToMany(transformation.BackwardFromContext()).
	Then(transformation.HasPrefix(clusterDackBox.Bucket)).
	ThenMapEachToOne(transformation.StripPrefix(clusterDackBox.Bucket))
