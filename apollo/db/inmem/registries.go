package inmem

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/apollo/registries/types"
	registryTypes "bitbucket.org/stack-rox/apollo/apollo/registries/types"
)

type registryStore struct {
	registries    map[string]registryTypes.ImageRegistry
	registryMutex sync.Mutex

	persistent db.Storage
}

func newRegistryStore(persistent db.Storage) *registryStore {
	return &registryStore{
		registries: make(map[string]registryTypes.ImageRegistry),
		persistent: persistent,
	}
}

// AddRegistry adds a registry
func (s *registryStore) AddRegistry(name string, registry types.ImageRegistry) {
	s.registryMutex.Lock()
	defer s.registryMutex.Unlock()
	s.registries[name] = registry
}

// RemoveRegistry removes a registry
func (s *registryStore) RemoveRegistry(name string) {
	s.registryMutex.Lock()
	defer s.registryMutex.Unlock()
	delete(s.registries, name)
}

// GetRegistries retrieves all registries from the DB
func (s *registryStore) GetRegistries() map[string]types.ImageRegistry {
	s.registryMutex.Lock()
	defer s.registryMutex.Unlock()
	return s.registries
}
