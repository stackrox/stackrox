package datastore

import (
	"context"

	"github.com/stackrox/rox/central/user/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	accessSAC = sac.ForResource(resources.Access)
)

type dataStoreImpl struct {
	storage store.Store
}

func (ds *dataStoreImpl) GetUser(ctx context.Context, name string) (*storage.User, error) {
	if ok, err := accessSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return ds.storage.GetUser(name)
}

func (ds *dataStoreImpl) GetAllUsers(ctx context.Context) ([]*storage.User, error) {
	if ok, err := accessSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return ds.storage.GetAllUsers()
}

func (ds *dataStoreImpl) Upsert(ctx context.Context, user *storage.User) error {
	if ok, err := accessSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return ds.storage.Upsert(user)
}
