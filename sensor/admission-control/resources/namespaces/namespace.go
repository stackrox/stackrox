package namespaces

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/admission-control/resources/deployments"
	"github.com/stackrox/rox/sensor/admission-control/resources/pods"
)

// NewNamespaceStore returns new instance of NamespaceStore.
func NewNamespaceStore(deployments *deployments.DeploymentStore, pods *pods.PodStore) *NamespaceStore {
	return &NamespaceStore{
		deployments:              deployments,
		pods:                     pods,
		namespaceNamesToMetadata: make(map[string]namespaceMetadata),
	}
}

// NamespaceStore stores namespace metadata, and provides handles deployments and pod removal when namespace is removed.
type NamespaceStore struct {
	deployments *deployments.DeploymentStore
	pods        *pods.PodStore

	lock                     sync.RWMutex
	namespaceNamesToMetadata map[string]namespaceMetadata
}

type namespaceMetadata struct {
	id          string
	annotations map[string]string
}

// AddNamespace adds namespace metadata keyed by namespace name to the namespace store
// exported func for test only
func (n *NamespaceStore) AddNamespace(ns *storage.NamespaceMetadata) {
	n.lock.Lock()
	defer n.lock.Unlock()
	if ns == nil {
		return
	}
	n.namespaceNamesToMetadata[ns.GetName()] = namespaceMetadata{
		id:          ns.GetId(),
		annotations: ns.GetAnnotations(),
	}
}

// GetAnnotationsForNamespace gets all the annotations for the namespace from the in memory store
func (n *NamespaceStore) GetAnnotationsForNamespace(name string) map[string]string {
	n.lock.Lock()
	defer n.lock.Unlock()

	return n.namespaceNamesToMetadata[name].annotations
}

func (n *NamespaceStore) removeNamespace(ns *storage.NamespaceMetadata) {
	n.lock.Lock()
	defer n.lock.Unlock()

	delete(n.namespaceNamesToMetadata, ns.GetId())
}

// ProcessEvent processes namespace event.
func (n *NamespaceStore) ProcessEvent(action central.ResourceAction, obj interface{}) {
	ns, isNs := obj.(*storage.NamespaceMetadata)
	if !isNs {
		return
	}

	switch action {
	case central.ResourceAction_CREATE_RESOURCE, central.ResourceAction_UPDATE_RESOURCE, central.ResourceAction_SYNC_RESOURCE:
		n.AddNamespace(ns)
	case central.ResourceAction_REMOVE_RESOURCE:
		n.removeNamespace(ns)
		n.deployments.OnNamespaceDelete(ns.GetName())
		n.pods.OnNamespaceDelete(ns.GetName())
	}
}
