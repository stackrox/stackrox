package orchestratornamespaces

import (
	"github.com/stackrox/stackrox/pkg/kubernetes"
	"github.com/stackrox/stackrox/pkg/set"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once       sync.Once
	namespaces OrchestratorNamespaces
)

// OrchestratorNamespaces stores the set of orchestrator namespaces
type OrchestratorNamespaces struct {
	nsSet set.StringSet
	lock  sync.RWMutex
}

// Singleton creates a new OrchestratorNamespaces object
func Singleton() *OrchestratorNamespaces {
	once.Do(func() {
		namespaces = OrchestratorNamespaces{
			nsSet: set.NewStringSet(),
		}
	})
	return &namespaces
}

// AddNamespace adds a namespace to the set
func (n *OrchestratorNamespaces) AddNamespace(ns string) {
	n.lock.Lock()
	defer n.lock.Unlock()

	namespaces.nsSet.Add(ns)
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
