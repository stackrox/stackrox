package resources

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/sync"
)

// A namespace store stores a mapping of namespace names to their ids.
type namespaceStore struct {
	lock sync.RWMutex

	namespaceNamesToIDs map[string]string
}

func newNamespaceStore() *namespaceStore {
	return &namespaceStore{
		namespaceNamesToIDs: make(map[string]string),
	}
}

func (n *namespaceStore) addNamespace(ns *storage.NamespaceMetadata) {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.namespaceNamesToIDs[ns.GetName()] = ns.GetId()
}

func (n *namespaceStore) lookupNamespaceID(name string) (string, bool) {
	n.lock.RLock()
	defer n.lock.RUnlock()

	id, found := n.namespaceNamesToIDs[name]
	return id, found
}
