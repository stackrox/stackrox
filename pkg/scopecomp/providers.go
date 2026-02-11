package scopecomp

//go:generate mockgen -package mocks -destination mocks/label_providers.go github.com/stackrox/rox/pkg/scopecomp ClusterLabelProvider,NamespaceLabelProvider

// ClusterLabelProvider provides cluster labels for a given cluster ID.
type ClusterLabelProvider interface {
	GetClusterLabels(clusterID string) (map[string]string, error)
}

// NamespaceLabelProvider provides namespace labels for a given namespace ID.
type NamespaceLabelProvider interface {
	GetNamespaceLabels(namespaceID string) (map[string]string, error)
}
