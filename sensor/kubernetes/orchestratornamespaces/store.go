package orchestratornamespaces

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

// OrchestratorNamespaces stores the set of orchestrator namespaces
type OrchestratorNamespaces struct {
	nsSet set.StringSet
	lock  sync.RWMutex
}

// ReconcileDelete is called after Sensor reconnects with Central and receives its state hashes.
// Reconciliacion ensures that Sensor and Central have the same state by checking whether a given resource
// shall be deleted from Central.
func (n *OrchestratorNamespaces) ReconcileDelete(resType, resID string, resHash uint64) (string, error) {
	_, _, _ = resType, resID, resHash
	// TODO implement me
	return "", errors.New("Not implemented")
}

// NewOrchestratorNamespaces returns a new OrchestratorNamespaces store
func NewOrchestratorNamespaces() *OrchestratorNamespaces {
	return &OrchestratorNamespaces{
		nsSet: set.NewStringSet(),
	}
}

// Cleanup deletes all entries from store
func (n *OrchestratorNamespaces) Cleanup() {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.nsSet.Clear()
}

// AddNamespace adds a namespace to the set
func (n *OrchestratorNamespaces) AddNamespace(ns string) {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.nsSet.Add(ns)
}

// IsOrchestratorNamespace checks if a namespace is a orchestrator namespace or not
func (n *OrchestratorNamespaces) IsOrchestratorNamespace(ns string) bool {
	n.lock.RLock()
	defer n.lock.RUnlock()

	if n.nsSet.Contains(ns) {
		return true
	}
	return kubernetes.IsSystemNamespace(ns)
}
