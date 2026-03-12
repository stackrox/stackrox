package resources

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
)

// A namespace store stores a mapping of namespace names to their metadata.
type namespaceStore struct {
	lock sync.RWMutex

	namespaces   map[string]*storage.NamespaceMetadata
	namespacesByID map[string]*storage.NamespaceMetadata
}

func (n *namespaceStore) Cleanup() {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.namespaces = make(map[string]*storage.NamespaceMetadata)
	n.namespacesByID = make(map[string]*storage.NamespaceMetadata)
}

func newNamespaceStore() *namespaceStore {
	return &namespaceStore{
		namespaces:     make(map[string]*storage.NamespaceMetadata),
		namespacesByID: make(map[string]*storage.NamespaceMetadata),
	}
}

func (n *namespaceStore) addNamespace(ns *storage.NamespaceMetadata) {
	n.lock.Lock()
	defer n.lock.Unlock()

	cloned := ns.CloneVT()
	n.namespaces[ns.GetName()] = cloned
	n.namespacesByID[ns.GetId()] = cloned
}

func (n *namespaceStore) removeNamespace(ns *storage.NamespaceMetadata) {
	n.lock.Lock()
	defer n.lock.Unlock()

	delete(n.namespaces, ns.GetName())
	delete(n.namespacesByID, ns.GetId())
}

func (n *namespaceStore) lookupNamespaceID(name string) (string, bool) {
	n.lock.RLock()
	defer n.lock.RUnlock()

	metadata, found := n.namespaces[name]
	return metadata.GetId(), found
}

// LookupNamespaceLabelsByID returns the labels for the given namespace ID.
// This is used by the label provider interface for policy scope matching.
func (n *namespaceStore) LookupNamespaceLabelsByID(id string) (map[string]string, bool) {
	n.lock.RLock()
	defer n.lock.RUnlock()

	metadata, found := n.namespacesByID[id]
	if !found {
		return nil, false
	}
	return metadata.GetLabels(), true
}
