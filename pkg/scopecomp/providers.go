package scopecomp

// ClusterLabelProvider provides cluster labels for a given cluster ID.
type ClusterLabelProvider interface {
	GetClusterLabels(clusterID string) (map[string]string, error)
}

// NamespaceLabelProvider provides namespace labels for a given namespace name.
type NamespaceLabelProvider interface {
	GetNamespaceLabels(namespaceName string) (map[string]string, error)
}
