package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

// NewNamespaceStore returns new instance of NamespaceStore.
func NewNamespaceStore(deployments *DeploymentStore, pods *PodStore) *NamespaceStore {
	return &NamespaceStore{
		deployments: deployments,
		pods:        pods,
	}
}

// NamespaceStore does not actually store namespaces, however provides handles deployments and pod removal when namespace is removed.
type NamespaceStore struct {
	deployments *DeploymentStore
	pods        *PodStore
}

// ProcessEvent processes namespace event.
func (m *NamespaceStore) ProcessEvent(action central.ResourceAction, obj interface{}) {
	ns, isNs := obj.(*storage.NamespaceMetadata)
	if !isNs {
		return
	}

	switch action {
	case central.ResourceAction_CREATE_RESOURCE, central.ResourceAction_UPDATE_RESOURCE, central.ResourceAction_SYNC_RESOURCE:
		return
	case central.ResourceAction_REMOVE_RESOURCE:
		// Namespace remove event contains full namespace metadata.
		m.deployments.OnNamespaceDelete(ns.GetName())
		m.pods.OnNamespaceDelete(ns.GetName())
	}
}
