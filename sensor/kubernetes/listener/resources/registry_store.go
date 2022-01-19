package resources

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/docker/types"
	"github.com/stackrox/rox/pkg/registries"
	dockerFactory "github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/tlscheck"
)

// RegistryStore stores cluster-internal registries by namespace.
type RegistryStore struct {
	factory registries.Factory
	// store maps a namespace to the names of registries accessible from within the namespace.
	// The registry maps to its credentials.
	store map[string]registries.Set

	mutex sync.RWMutex
}

// newRegistryStore creates a new registryStore.
func newRegistryStore() *RegistryStore {
	return &RegistryStore{
		factory: registries.NewFactory(registries.WithRegistryCreators(dockerFactory.Creator)),
		store:   make(map[string]registries.Set),
	}
}

func (rs *RegistryStore) addOrUpdateRegistry(namespace, registry string, dce types.DockerConfigEntry) {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	regs := rs.store[namespace]
	if regs == nil {
		regs = registries.NewSet(rs.factory)
		rs.store[namespace] = regs
	}

	tlscheck.CheckTLS(registry)
	regs.UpdateImageIntegration(&storage.ImageIntegration{
		Name:                 registry,
		Type:                 "docker",
		Categories:           []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		IntegrationConfig:    &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint:             registry,
				Username:             dce.Username,
				Password:             dce.Password,
				Insecure:             false,
			},
		},
	})
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
