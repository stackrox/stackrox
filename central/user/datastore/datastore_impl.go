package datastore

import (
	"context"

	"github.com/stackrox/rox/central/user/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
)

type dataStoreImpl struct {
	storage store.Store
}

func (ds *dataStoreImpl) GetUser(ctx context.Context, name string) (*storage.User, error) {
	return ds.storage.GetUser(name)
}

func (ds *dataStoreImpl) GetAllUsers(ctx context.Context) ([]*storage.User, error) {
	return ds.storage.GetAllUsers()
}

func (ds *dataStoreImpl) Upsert(ctx context.Context, user *storage.User) error {
	return ds.storage.Upsert(user)
}
