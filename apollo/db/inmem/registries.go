package inmem

import (
	"bitbucket.org/stack-rox/apollo/apollo/registries/types"
)

// AddRegistry adds a registry
func (i *InMemoryStore) AddRegistry(name string, registry types.ImageRegistry) {
	i.registryMutex.Lock()
	defer i.registryMutex.Unlock()
	i.registries[name] = registry
}

// RemoveRegistry removes a registry
func (i *InMemoryStore) RemoveRegistry(name string) {
	i.registryMutex.Lock()
	defer i.registryMutex.Unlock()
	delete(i.registries, name)
}

// GetRegistries retrieves all registries from the DB
func (i *InMemoryStore) GetRegistries() map[string]types.ImageRegistry {
	i.registryMutex.Lock()
	defer i.registryMutex.Unlock()
	return i.registries
}
