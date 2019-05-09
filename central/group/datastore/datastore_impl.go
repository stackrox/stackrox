package datastore

import (
	"context"

	"github.com/stackrox/rox/central/group/store"
	"github.com/stackrox/rox/generated/storage"
)

type dataStoreImpl struct {
	storage store.Store
}

func (ds *dataStoreImpl) Get(ctx context.Context, props *storage.GroupProperties) (*storage.Group, error) {
	return ds.storage.Get(props)
}

func (ds *dataStoreImpl) GetAll(ctx context.Context) ([]*storage.Group, error) {
	return ds.storage.GetAll()
}

func (ds *dataStoreImpl) Walk(ctx context.Context, authProviderID string, attributes map[string][]string) ([]*storage.Group, error) {
	return ds.storage.Walk(authProviderID, attributes)
}

func (ds *dataStoreImpl) Add(ctx context.Context, group *storage.Group) error {
	return ds.storage.Add(group)
}

func (ds *dataStoreImpl) Update(ctx context.Context, group *storage.Group) error {
	return ds.storage.Update(group)
}

func (ds *dataStoreImpl) Upsert(ctx context.Context, group *storage.Group) error {
	return ds.storage.Upsert(group)
}

func (ds *dataStoreImpl) Mutate(ctx context.Context, remove, update, add []*storage.Group) error {
	return ds.storage.Mutate(remove, update, add)
}

func (ds *dataStoreImpl) Remove(ctx context.Context, props *storage.GroupProperties) error {
	return ds.storage.Remove(props)
}
