package inmem

import (
	"fmt"
	"sort"
	"sync"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type registryStore struct {
	registries    map[string]*v1.Registry
	registryMutex sync.Mutex

	persistent db.RegistryStorage
}

func newRegistryStore(persistent db.RegistryStorage) *registryStore {
	return &registryStore{
		registries: make(map[string]*v1.Registry),
		persistent: persistent,
	}
}

func (s *registryStore) loadFromPersistent() error {
	s.registryMutex.Lock()
	defer s.registryMutex.Unlock()
	registries, err := s.persistent.GetRegistries(&v1.GetRegistriesRequest{})
	if err != nil {
		return err
	}
	for _, registry := range registries {
		s.registries[registry.Name] = registry
	}
	return nil
}

// GetRegistry returns a registry, if it exists or an error based on the name parameter
func (s *registryStore) GetRegistry(name string) (registry *v1.Registry, exists bool, err error) {
	s.registryMutex.Lock()
	defer s.registryMutex.Unlock()
	registry, exists = s.registries[name]
	return
}

func (s *registryStore) GetRegistries(request *v1.GetRegistriesRequest) ([]*v1.Registry, error) {
	s.registryMutex.Lock()
	defer s.registryMutex.Unlock()
	registrySlice := make([]*v1.Registry, 0, len(s.registries))
	for _, registry := range s.registries {
		registrySlice = append(registrySlice, registry)
	}
	sort.SliceStable(registrySlice, func(i, j int) bool { return registrySlice[i].Name < registrySlice[j].Name })
	return registrySlice, nil
}

func (s *registryStore) upsertRegistry(registry *v1.Registry) {
	s.registryMutex.Lock()
	defer s.registryMutex.Unlock()
	s.registries[registry.Name] = registry
}

// AddRegistry upserts a registry
func (s *registryStore) AddRegistry(registry *v1.Registry) error {
	s.registryMutex.Lock()
	if _, exists := s.registries[registry.Name]; exists {
		s.registryMutex.Unlock()
		return fmt.Errorf("Registry with name %v already exists", registry.Name)
	}
	s.registryMutex.Unlock()
	if err := s.persistent.AddRegistry(registry); err != nil {
		return err
	}
	s.upsertRegistry(registry)
	return nil
}

// UpdateRegistry upserts a registry
func (s *registryStore) UpdateRegistry(registry *v1.Registry) error {
	if err := s.persistent.UpdateRegistry(registry); err != nil {
		return err
	}
	s.upsertRegistry(registry)
	return nil
}

// RemoveRegistry removes a registry
func (s *registryStore) RemoveRegistry(name string) error {
	if err := s.persistent.RemoveRegistry(name); err != nil {
		return err
	}
	s.registryMutex.Lock()
	defer s.registryMutex.Unlock()
	delete(s.registries, name)
	return nil
}
