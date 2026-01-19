package resources

import (
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

	n.namespaces[ns.GetName()] = ns.CloneVT()
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

// LookupNamespaceLabels returns the labels for the given namespace.
func (n *namespaceStore) LookupNamespaceLabels(name string) (map[string]string, bool) {
	n.lock.RLock()
	defer n.lock.RUnlock()

	metadata, found := n.namespaces[name]
	return metadata.GetLabels(), found
}
