package namespaces

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
)

// NamespaceStore is a namespace store stores a mapping of namespace names to their ids.
type NamespaceStore struct {
	lock sync.RWMutex

	namespaceNamesToMetadata map[string]namespaceMetadata
}

type namespaceMetadata struct {
	id         string
	annotation map[string]string
}

func newNamespaceStore() *NamespaceStore {
	return &NamespaceStore{
		namespaceNamesToMetadata: make(map[string]namespaceMetadata),
	}
}

func (n *NamespaceStore) AddNamespace(ns *storage.NamespaceMetadata) {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.namespaceNamesToMetadata[ns.GetName()] = namespaceMetadata{
		id:         ns.GetId(),
		annotation: ns.GetAnnotations(),
	}
}

func (n *NamespaceStore) GetAnnotationsForNamespace(name string) map[string]string {
	n.lock.RLock()
	defer n.lock.RUnlock()

	return n.namespaceNamesToMetadata[name].annotation
}

func (n *NamespaceStore) LookupNamespaceID(name string) (string, bool) {
	n.lock.RLock()
	defer n.lock.RUnlock()

	metadata, found := n.namespaceNamesToMetadata[name]
	if found {
		return metadata.id, found
	}
	return "", found
}
