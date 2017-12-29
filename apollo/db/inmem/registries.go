package inmem

import (
	"fmt"
	"sort"
	"sync"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/golang/protobuf/proto"
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

func (s *registryStore) clone(registry *v1.Registry) *v1.Registry {
	return proto.Clone(registry).(*v1.Registry)
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
	return s.clone(registry), exists, nil
}

func (s *registryStore) GetRegistries(request *v1.GetRegistriesRequest) ([]*v1.Registry, error) {
	s.registryMutex.Lock()
	defer s.registryMutex.Unlock()
	registrySlice := make([]*v1.Registry, 0, len(s.registries))
	for _, registry := range s.registries {
		if len(request.GetCluster()) != 0 && !sliceContains(registry.GetClusters(), request.GetCluster()) {
			continue
		}
		registrySlice = append(registrySlice, s.clone(registry))
	}
	sort.SliceStable(registrySlice, func(i, j int) bool { return registrySlice[i].Name < registrySlice[j].Name })
	return registrySlice, nil
}

func (s *registryStore) upsertRegistry(registry *v1.Registry) {
	s.registries[registry.Name] = s.clone(registry)
}

// AddRegistry upserts a registry
func (s *registryStore) AddRegistry(registry *v1.Registry) error {
	s.registryMutex.Lock()
	defer s.registryMutex.Unlock()
	if _, exists := s.registries[registry.Name]; exists {
		return fmt.Errorf("Registry with name %v already exists", registry.Name)
	}
	if err := s.persistent.AddRegistry(registry); err != nil {
		return err
	}
	s.upsertRegistry(registry)
	return nil
}

// UpdateRegistry upserts a registry
func (s *registryStore) UpdateRegistry(registry *v1.Registry) error {
	s.registryMutex.Lock()
	defer s.registryMutex.Unlock()
	if err := s.persistent.UpdateRegistry(registry); err != nil {
		return err
	}
	s.upsertRegistry(registry)
	return nil
}

// RemoveRegistry removes a registry
func (s *registryStore) RemoveRegistry(name string) error {
	s.registryMutex.Lock()
	defer s.registryMutex.Unlock()
	if err := s.persistent.RemoveRegistry(name); err != nil {
		return err
	}
	delete(s.registries, name)
	return nil
}
