package resources

import (
	"github.com/stackrox/rox/generated/storage"
)

// A namespace store stores a mapping of namespace names to their ids.
type namespaceStore struct {
	namespaceNamesToIDs map[string]string
}

func newNamespaceStore() *namespaceStore {
	return &namespaceStore{
		namespaceNamesToIDs: make(map[string]string),
	}
}

func (n *namespaceStore) addNamespace(ns *storage.NamespaceMetadata) {
	n.namespaceNamesToIDs[ns.GetName()] = ns.GetId()
}

func (n *namespaceStore) lookupNamespaceID(name string) (string, bool) {
	id, found := n.namespaceNamesToIDs[name]
	return id, found
}
