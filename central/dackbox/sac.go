package dackbox

import (
	"context"

	clusterDackBox "github.com/stackrox/rox/central/cluster/dackbox"
	cveDackBox "github.com/stackrox/rox/central/cve/dackbox"
	"github.com/stackrox/rox/central/idmap"
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

	// NodeSACTransform transforms node ids into their SAC scopes.
	NodeSACTransform = clusterScoped(NodeTransformations[v1.SearchCategory_CLUSTERS])

	// NodeComponentEdgeSACTransform transforms node:component edge ids into their SAC scopes.
	NodeComponentEdgeSACTransform = clusterScoped(NodeComponentEdgeTransformations[v1.SearchCategory_CLUSTERS])

	// NodeCVEEdgeSACTransform transforms node:cve edge ids into their SAC scopes.
	NodeCVEEdgeSACTransform = clusterScoped(NodeCVEEdgeTransformations[v1.SearchCategory_CLUSTERS])

	// DeploymentSACTransform transforms deployment ids into their SAC scopes.
	DeploymentSACTransform = namespaceScoped(DeploymentTransformations[v1.SearchCategory_NAMESPACES])

	// ImageSACTransform transforms image ids into their SAC scopes.
	ImageSACTransform = namespaceScoped(ImageTransformations[v1.SearchCategory_NAMESPACES])

	// ImageComponentEdgeSACTransform transforms image:component edge ids into their SAC scopes.
	ImageComponentEdgeSACTransform = namespaceScoped(ImageComponentEdgeTransformations[v1.SearchCategory_NAMESPACES])

	// ImageCVEEdgeSACTransform transforms image:cve edge ids into their SAC scopes.
	ImageCVEEdgeSACTransform = namespaceScoped(ImageCVEEdgeTransformations[v1.SearchCategory_NAMESPACES])

	// ImageComponentSACTransform transforms image component ids into their SAC scopes.
	ImageComponentSACTransform = namespaceScoped(ComponentTransformations[v1.SearchCategory_NAMESPACES])

	// NodeComponentSACTransform transforms node component ids into their SAC scopes.
	NodeComponentSACTransform = clusterScoped(componentNodeClusterSACTransformation)

	// ComponentVulnEdgeSACTransform transforms component:vulnerability edge ids into their SAC scopes.
	ComponentVulnEdgeSACTransform = namespaceScoped(ComponentCVEEdgeTransformations[v1.SearchCategory_NAMESPACES])

	// ImageVulnSACTransform transforms image component vulnerability ids into their SAC scopes.
	ImageVulnSACTransform = namespaceScoped(CVETransformations[v1.SearchCategory_NAMESPACES])

	// NodeVulnSACTransform transforms node component vulnerability ids into their SAC scopes.
	NodeVulnSACTransform = clusterScoped(cveNodeClusterSACTransformation)

	// ClusterVulnEdgeSACTransform transforms cluster:vulnerability edge ids into their SAC scopes.
	ClusterVulnEdgeSACTransform = clusterScoped(cveToClustersWithoutDeploymentsNorNodes)

	// ClusterVulnSACTransform transforms cluster vulnerability ids into their SAC scopes.
	ClusterVulnSACTransform = clusterScoped(cveToClustersWithoutDeploymentsNorNodes)
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
	idMap := idmap.FromContext(ctx)

	ret := make([][]sac.ScopeKey, 0, len(namespaceIDs))
	for _, namespaceID := range namespaceIDs {
		nsInfo := idMap.ByNamespaceID(string(namespaceID))
		if nsInfo == nil {
			// If we can't find the namespace info, conservatively check for any/any. This will prevent information
			// leakage, while not impacted users with sufficient (global) privileges.
			ret = append(ret, []sac.ScopeKey{sac.ClusterScopeKey(""), sac.NamespaceScopeKey("")})
			continue
		}

		ret = append(ret, []sac.ScopeKey{sac.ClusterScopeKey(nsInfo.ClusterID), sac.NamespaceScopeKey(nsInfo.Name)})
	}
	return ret
}

var cveToClustersWithoutDeploymentsNorNodes = transformation.AddPrefix(cveDackBox.Bucket).
	ThenMapToMany(transformation.BackwardFromContext(clusterDackBox.Bucket)).
	ThenMapEachToOne(transformation.StripPrefixUnchecked(clusterDackBox.Bucket))
