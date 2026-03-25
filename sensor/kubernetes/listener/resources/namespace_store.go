package resources

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
)

// A namespace store stores a mapping of namespace names to their metadata.
type namespaceStore struct {
	lock sync.RWMutex

	namespaces map[string]*storage.NamespaceMetadata
}

func (n *namespaceStore) Cleanup() {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.namespaces = make(map[string]*storage.NamespaceMetadata)
}

func newNamespaceStore() *namespaceStore {
	return &namespaceStore{
		namespaces: make(map[string]*storage.NamespaceMetadata),
	}
}

func (n *namespaceStore) addNamespace(ns *storage.NamespaceMetadata) {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.namespaces[ns.GetName()] = ns
}

func (n *namespaceStore) removeNamespace(ns *storage.NamespaceMetadata) {
	n.lock.Lock()
	defer n.lock.Unlock()

	delete(n.namespaces, ns.GetName())
}

func (n *namespaceStore) lookupNamespaceID(name string) (string, bool) {
	n.lock.RLock()
	defer n.lock.RUnlock()

	metadata, found := n.namespaces[name]
	return metadata.GetId(), found
}

// GetNamespaceLabels returns the labels for the given namespace name.
// This is used by the label provider interface for policy scope matching.
// The clusterID parameter is unused since Sensor only manages one cluster.
func (n *namespaceStore) GetNamespaceLabels(_ context.Context, _ string, namespaceName string) (map[string]string, error) {
	n.lock.RLock()
	defer n.lock.RUnlock()

	metadata, found := n.namespaces[namespaceName]
	if !found {
		log.Debugf("Namespace %q not found in store, labels unavailable for policy evaluation", namespaceName)
		return nil, nil
	}
	return metadata.GetLabels(), nil
}

// GetAll returns all namespace metadata.
func (n *namespaceStore) GetAll() []*storage.NamespaceMetadata {
	n.lock.RLock()
	defer n.lock.RUnlock()

	var ret []*storage.NamespaceMetadata
	for _, ns := range n.namespaces {
		ret = append(ret, ns)
	}
	return ret
}
