package inmem

import (
	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type registryStore struct {
	db.RegistryStorage
}

func newRegistryStore(persistent db.RegistryStorage) *registryStore {
	return &registryStore{
		RegistryStorage: persistent,
	}
}

func (s *registryStore) GetRegistries(request *v1.GetRegistriesRequest) ([]*v1.Registry, error) {
	registries, err := s.RegistryStorage.GetRegistries(request)
	if err != nil {
		return nil, err
	}
	registrySlice := registries[:0]
	for _, registry := range registries {
		if len(request.GetCluster()) != 0 && !sliceContains(registry.GetClusters(), request.GetCluster()) {
			continue
		}
		registrySlice = append(registrySlice, registry)
	}
	return registrySlice, nil
}
