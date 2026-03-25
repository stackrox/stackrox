package resources

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()
)

// NewNamespaceStore returns new instance of NamespaceStore.
func NewNamespaceStore(deployments *DeploymentStore, pods *PodStore) *NamespaceStore {
	return &NamespaceStore{
		namespaces:  make(map[string]*storage.NamespaceMetadata),
		deployments: deployments,
		pods:        pods,
	}
}

// NamespaceStore stores namespace metadata and handles deployments and pod removal when namespace is removed.
type NamespaceStore struct {
	lock        sync.RWMutex
	namespaces  map[string]*storage.NamespaceMetadata // keyed by namespace name
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
		m.lock.Lock()
		defer m.lock.Unlock()
		m.namespaces[ns.GetName()] = ns
	case central.ResourceAction_REMOVE_RESOURCE:
		// Namespace remove event contains full namespace metadata.
		m.lock.Lock()
		defer m.lock.Unlock()
		delete(m.namespaces, ns.GetName())
		m.deployments.OnNamespaceDelete(ns.GetName())
		m.pods.OnNamespaceDelete(ns.GetName())
	}
}

// GetNamespaceLabels returns the labels for the given namespace name.
// This is used by the label provider interface for policy scope matching.
// The clusterID parameter is unused since admission control only manages one cluster.
func (m *NamespaceStore) GetNamespaceLabels(_ context.Context, _ string, namespaceName string) (map[string]string, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	metadata, found := m.namespaces[namespaceName]
	if !found {
		log.Debugf("Namespace %q not found in store, labels unavailable for policy evaluation", namespaceName)
		return nil, nil
	}
	return metadata.GetLabels(), nil
}
