package resources

import (
	"github.com/stackrox/rox/pkg/docker/types"
	"github.com/stackrox/rox/pkg/sync"
)

// RegistryStore stores cluster-internal registries by namespace.
type RegistryStore struct {
	// store maps a namespace to the names of registries accessible from within the namespace.
	// The registry maps to its credentials.
	store map[string]map[string]types.DockerConfigEntry

	mutex sync.RWMutex
}

// newRegistryStore creates a new registryStore.
func newRegistryStore() *RegistryStore {
	return &RegistryStore{
		store: make(map[string]map[string]types.DockerConfigEntry),
	}
}

func (rs *RegistryStore) addOrUpdateRegistry(namespace, registry string, dce types.DockerConfigEntry) {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	nsMap := rs.store[namespace]
	if nsMap == nil {
		nsMap = make(map[string]types.DockerConfigEntry)
		rs.store[namespace] = nsMap
	}

	nsMap[registry] = dce
}

// getAllInNamespace returns all the registries+credentials within a given namespace.
func (rs *RegistryStore) getAllInNamespace(namespace string) map[string]types.DockerConfigEntry {
	regs := make(map[string]types.DockerConfigEntry)

	rs.mutex.RLock()
	rs.mutex.RUnlock()

	// Copy the registry to configuration map.
	for reg, dce := range rs.store[namespace] {
		regs[reg] = dce
	}

	return regs
}
