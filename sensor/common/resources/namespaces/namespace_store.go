package namespaces

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
)

// NamespaceStore is a namespace store stores a mapping of namespace names to their ids.
type NamespaceStore struct {
	lock sync.RWMutex

	namespaceNamesToMetadata map[string]namespaceMetadata
}

type namespaceMetadata struct {
	id          string
	annotations map[string]string
}

func newNamespaceStore() *NamespaceStore {
	return &NamespaceStore{
		namespaceNamesToMetadata: make(map[string]namespaceMetadata),
	}
}

// NewTestNamespaceStore returns a namespace store for testing purposes
func NewTestNamespaceStore(t *testing.T) *NamespaceStore {
	if t == nil {
		return nil
	}
	return newNamespaceStore()
}

// AddNamespace adds a namespace to the datastore.
func (n *NamespaceStore) AddNamespace(ns *storage.NamespaceMetadata) {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.namespaceNamesToMetadata[ns.GetName()] = namespaceMetadata{
		id:          ns.GetId(),
		annotations: ns.GetAnnotations(),
	}
}

// GetAnnotationsForNamespace returns the annotations for the given namespace.
func (n *NamespaceStore) GetAnnotationsForNamespace(name string) map[string]string {
	n.lock.RLock()
	defer n.lock.RUnlock()

	return n.namespaceNamesToMetadata[name].annotations
}

// LookupNamespaceID returns the ID of a given namespace if it exists.
func (n *NamespaceStore) LookupNamespaceID(name string) (string, bool) {
	n.lock.RLock()
	defer n.lock.RUnlock()

	metadata, found := n.namespaceNamesToMetadata[name]
	if found {
		return metadata.id, found
	}
	return "", found
}
