package resources

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/deduper"
)

// A namespace store stores a mapping of namespace names to their ids.
type namespaceStore struct {
	lock sync.RWMutex

	namespaceNamesToIDs map[string]string
}

func (n *namespaceStore) Cleanup() {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.namespaceNamesToIDs = make(map[string]string)
}

func (n *namespaceStore) ReconcileDelete(resType, resID string, _ uint64) (string, error) {
	if resType != deduper.TypeNamespace.String() {
		return "", errors.Errorf("resource type %s not supported", resType)
	}
	n.lock.Lock()
	defer n.lock.Unlock()
	for _, id := range n.namespaceNamesToIDs {
		if id == resID {
			return "", nil
		}
	}
	return resID, nil
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

func (n *namespaceStore) removeNamespace(ns *storage.NamespaceMetadata) {
	n.lock.Lock()
	defer n.lock.Unlock()

	delete(n.namespaceNamesToIDs, ns.GetName())
}

func (n *namespaceStore) lookupNamespaceID(name string) (string, bool) {
	n.lock.RLock()
	defer n.lock.RUnlock()

	id, found := n.namespaceNamesToIDs[name]
	return id, found
}
