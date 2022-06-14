package dackbox

import (
	"context"

	"github.com/pkg/errors"
	clusterDackBox "github.com/stackrox/rox/central/cluster/dackbox"
	"github.com/stackrox/rox/central/idmap"
	nsDackBox "github.com/stackrox/rox/central/namespace/dackbox"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search/filtered"
)

var log = logging.LoggerForModule()

var (
	// ClusterSACTransform transforms cluster ids into their SAC scopes.
	ClusterSACTransform = clusterScoped(ClusterTransformationPaths[v1.SearchCategory_CLUSTERS])

	// NamespaceSACTransform transforms namespace ids into their SAC scopes.
	NamespaceSACTransform = namespaceScoped(NamespaceTransformationPaths[v1.SearchCategory_NAMESPACES])

	// NodeSACTransform transforms node ids into their SAC scopes.
	NodeSACTransform = clusterScoped(NodeTransformationPaths[v1.SearchCategory_CLUSTERS])

	// NodeComponentEdgeSACTransform transforms node:component edge ids into their SAC scopes.
	NodeComponentEdgeSACTransform = fromEdgeSource(clusterScoped(NodeTransformationPaths[v1.SearchCategory_CLUSTERS]))

	// NodeCVEEdgeSACTransform transforms node:cve edge ids into their SAC scopes.
	NodeCVEEdgeSACTransform = fromEdgeSource(clusterScoped(NodeTransformationPaths[v1.SearchCategory_CLUSTERS]))

	// DeploymentSACTransform transforms deployment ids into their SAC scopes.
	DeploymentSACTransform = namespaceScoped(DeploymentTransformationPaths[v1.SearchCategory_NAMESPACES])

	// ActiveComponentSACTransform transforms active component into their SAC scopes.
	ActiveComponentSACTransform = namespaceScoped(ActiveComponentTransformationPaths[v1.SearchCategory_NAMESPACES])

	// ImageSACTransform transforms image ids into their SAC scopes.
	ImageSACTransform = namespaceScoped(ImageTransformationPaths[v1.SearchCategory_NAMESPACES])

	// ImageComponentEdgeSACTransform transforms image:component edge ids into their SAC scopes.
	ImageComponentEdgeSACTransform = fromEdgeSource(namespaceScoped(ImageTransformationPaths[v1.SearchCategory_NAMESPACES]))

	// ImageCVEEdgeSACTransform transforms image:cve edge ids into their SAC scopes.
	ImageCVEEdgeSACTransform = fromEdgeSource(namespaceScoped(ImageTransformationPaths[v1.SearchCategory_NAMESPACES]))

	// ComponentVulnEdgeSACTransform transforms component:vulnerability edge ids into their SAC scopes.
	ComponentVulnEdgeSACTransform = fromEdgeSource(namespaceScoped(ComponentTransformationPaths[v1.SearchCategory_NAMESPACES]))

	// ClusterVulnEdgeSACTransform transforms cluster:vulnerability edge ids into their SAC scopes.
	ClusterVulnEdgeSACTransform = fromEdgeSource(clusterScoped(ClusterTransformationPaths[v1.SearchCategory_CLUSTERS]))
)

func clusterScoped(toClusterIDsPath dackbox.BucketPath) filtered.ScopeTransform {
	if _, err := dackbox.ConcatenatePaths(toClusterIDsPath, dackbox.BackwardsBucketPath(clusterDackBox.BucketHandler)); err != nil {
		panic(errors.Wrap(err, "invalid path for cluster-scoped SAC transform, must end with cluster bucket"))
	}
	return filtered.ScopeTransform{
		Path:      toClusterIDsPath,
		ScopeFunc: clusterIDToScope,
	}
}

func clusterIDToScope(_ context.Context, clusterID string) []sac.ScopeKey {
	return []sac.ScopeKey{sac.ClusterScopeKey(clusterID)}
}

func namespaceScoped(toNamespaceIDsPath dackbox.BucketPath) filtered.ScopeTransform {
	if _, err := dackbox.ConcatenatePaths(toNamespaceIDsPath, dackbox.BackwardsBucketPath(nsDackBox.BucketHandler)); err != nil {
		panic(errors.Wrap(err, "invalid path for namespace-scoped SAC transform, must end with namespace bucket"))
	}
	return filtered.ScopeTransform{
		Path:      toNamespaceIDsPath,
		ScopeFunc: namespaceIDToScope,
	}
}

// fromEdgeSource modifies the given ScopeTransform to treat the input ID as an edge key, and operate on the first
// (source) component.
func fromEdgeSource(transform filtered.ScopeTransform) filtered.ScopeTransform {
	if transform.Path.Len() > 1 && !transform.Path.BackwardTraversal {
		panic(errors.New("path in scope transform from edge source must be a backwards path"))
	}
	transformFromEdge := transform
	transformFromEdge.EdgeIndex = pointers.Int(0)
	return transformFromEdge
}

func namespaceIDToScope(ctx context.Context, namespaceID string) []sac.ScopeKey {
	idMap := idmap.FromContext(ctx)

	nsInfo := idMap.ByNamespaceID(string(namespaceID))
	if nsInfo == nil {
		// If we can't find the namespace info, conservatively require any/any access. This will prevent information
		// leakage, while not impacted users with sufficient (global) privileges.
		return []sac.ScopeKey{sac.ClusterScopeKey(""), sac.NamespaceScopeKey("")}
	}

	return []sac.ScopeKey{sac.ClusterScopeKey(nsInfo.ClusterID), sac.NamespaceScopeKey(nsInfo.Name)}
}
