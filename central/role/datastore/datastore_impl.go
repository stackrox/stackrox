package datastore

import (
	"context"

	"github.com/stackrox/rox/central/role/store"
	"github.com/stackrox/rox/generated/storage"
)

type dataStoreImpl struct {
	storage store.Store
}

func (ds *dataStoreImpl) GetRole(ctx context.Context, name string) (*storage.Role, error) {
	return ds.storage.GetRole(name)
}

func (ds *dataStoreImpl) GetRolesBatch(ctx context.Context, names []string) ([]*storage.Role, error) {
	return ds.storage.GetRolesBatch(names)
}

func (ds *dataStoreImpl) GetAllRoles(ctx context.Context) ([]*storage.Role, error) {
	return ds.storage.GetAllRoles()
}

func (ds *dataStoreImpl) AddRole(ctx context.Context, role *storage.Role) error {
	return ds.storage.AddRole(role)
}

func (ds *dataStoreImpl) UpdateRole(ctx context.Context, role *storage.Role) error {
	return ds.storage.UpdateRole(role)
}

func (ds *dataStoreImpl) RemoveRole(ctx context.Context, name string) error {
	return ds.storage.RemoveRole(name)
}
