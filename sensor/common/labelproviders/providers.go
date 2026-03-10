package labelproviders

import (
	"context"

	"github.com/stackrox/rox/sensor/common/clusterlabels"
)

// ClusterLabelProviderAdapter adapts Sensor's cluster labels store to the ClusterLabelProvider interface.
type ClusterLabelProviderAdapter struct {
	store *clusterlabels.Store
}

// NewClusterLabelProviderAdapter creates a new cluster label provider adapter.
func NewClusterLabelProviderAdapter(store *clusterlabels.Store) *ClusterLabelProviderAdapter {
	return &ClusterLabelProviderAdapter{store: store}
}

// GetClusterLabels returns cluster labels. The clusterID parameter is ignored since Sensor
// only manages a single cluster.
func (a *ClusterLabelProviderAdapter) GetClusterLabels(_ context.Context, _ string) (map[string]string, error) {
	return a.store.Get(), nil
}

// NamespaceLabelProviderAdapter adapts Sensor's namespace store to the NamespaceLabelProvider interface.
type NamespaceLabelProviderAdapter struct {
	lookupFunc func(string) (map[string]string, bool)
}

// NewNamespaceLabelProviderAdapter creates a new namespace label provider adapter.
// The lookupFunc should be a function that takes a namespace ID and returns labels.
func NewNamespaceLabelProviderAdapter(lookupFunc func(string) (map[string]string, bool)) *NamespaceLabelProviderAdapter {
	return &NamespaceLabelProviderAdapter{lookupFunc: lookupFunc}
}

// GetNamespaceLabels returns labels for the given namespace ID.
func (a *NamespaceLabelProviderAdapter) GetNamespaceLabels(_ context.Context, namespaceID string) (map[string]string, error) {
	labels, found := a.lookupFunc(namespaceID)
	if !found {
		return nil, nil
	}
	return labels, nil
}
