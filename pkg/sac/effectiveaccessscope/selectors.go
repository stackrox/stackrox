package effectiveaccessscope

import (
	"github.com/stackrox/rox/pkg/set"
	"k8s.io/apimachinery/pkg/labels"
)

type selectors struct {
	clustersByID    set.StringSet
	clustersByName  set.StringSet
	clustersByLabel []labels.Selector

	namespacesByClusterID   map[string]set.StringSet
	namespacesByClusterName map[string]set.StringSet
	namespacesByLabel       []labels.Selector
}

func (s *selectors) matchCluster(cluster Cluster) scopeState {
	if s == nil {
		return Excluded
	}

	clusterID := cluster.GetId()
	if s.clustersByID != nil && s.clustersByID.Contains(clusterID) {
		return Included
	}

	clusterName := cluster.GetName()
	if s.clustersByName != nil && s.clustersByName.Contains(clusterName) {
		return Included
	}

	// Augment cluster labels with cluster's name.
	augmentedClusterLabels := augmentLabels(cluster.GetLabels(), clusterNameLabel, clusterName)
	return matchLabels(s.clustersByLabel, augmentedClusterLabels)
}

func (s *selectors) matchNamespace(namespace Namespace) scopeState {
	if s == nil {
		return Excluded
	}

	namespaceName := namespace.GetName()

	clusterID := namespace.GetClusterId()
	if clusterID != "" && matchNamespaceByClusterKey(s.namespacesByClusterID, clusterID, namespaceName) {
		return Included
	}

	clusterName := namespace.GetClusterName()
	if clusterName != "" && matchNamespaceByClusterKey(s.namespacesByClusterName, clusterName, namespaceName) {
		return Included
	}

	// Augment namespace labels with namespace's FQSN.
	namespaceFQSN := getNamespaceFQSN(clusterName, namespaceName)
	namespaceLabels := augmentLabels(namespace.GetLabels(), namespaceNameLabel, namespaceFQSN)

	return matchLabels(s.namespacesByLabel, namespaceLabels)
}

func matchNamespaceByClusterKey(targetMap map[string]set.StringSet, clusterKey string, namespaceKey string) bool {
	if targetMap == nil {
		return false
	}
	if _, clusterFound := targetMap[clusterKey]; !clusterFound {
		return false
	}
	clusterNamespaces := targetMap[clusterKey]
	return clusterNamespaces.Contains(namespaceKey)
}
