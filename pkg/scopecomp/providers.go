package scopecomp

import "context"

//go:generate mockgen -package mocks -destination mocks/label_providers.go github.com/stackrox/rox/pkg/scopecomp ClusterLabelProvider,NamespaceLabelProvider

// ClusterLabelProvider provides cluster labels for a given cluster ID.
type ClusterLabelProvider interface {
	GetClusterLabels(ctx context.Context, clusterID string) (map[string]string, error)
}

// NamespaceLabelProvider provides namespace labels for a given namespace ID.
type NamespaceLabelProvider interface {
	GetNamespaceLabels(ctx context.Context, namespaceID string) (map[string]string, error)
}
