package service

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

func toStorageProto(sync *v1.PolicySync) *storage.PolicySync {
	return &storage.PolicySync{
		Name:       sync.GetName(),
		Registries: toStorageRegistries(sync.GetRegistries()),
	}
}

func toV1Proto(sync *storage.PolicySync) *v1.PolicySync {
	return &v1.PolicySync{
		Name:       sync.GetName(),
		Registries: toV1Registries(sync.GetRegistries()),
	}
}

func toV1Registries(registries []*storage.PolicySync_Registry) []*v1.PolicySync_Registry {
	v1Registries := make([]*v1.PolicySync_Registry, 0, len(registries))
	for _, registry := range registries {
		v1Registries = append(v1Registries, &v1.PolicySync_Registry{
			Hostname:   registry.GetHostname(),
			Repository: registry.GetRepository(),
		})
	}
	return v1Registries
}

func toStorageRegistries(registries []*v1.PolicySync_Registry) []*storage.PolicySync_Registry {
	storageRegistries := make([]*storage.PolicySync_Registry, 0, len(registries))
	for _, registry := range registries {
		storageRegistries = append(storageRegistries, &storage.PolicySync_Registry{
			Hostname:   registry.GetHostname(),
			Repository: registry.GetRepository(),
		})
	}
	return storageRegistries
}
