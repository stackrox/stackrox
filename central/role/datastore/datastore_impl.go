package datastore

import (
	"context"
	"errors"

	"github.com/stackrox/rox/central/role/datastore/internal/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	roleSAC = sac.ForResource(resources.Role)
)

type dataStoreImpl struct {
	storage store.Store
}

func (ds *dataStoreImpl) GetRole(ctx context.Context, name string) (*storage.Role, error) {
	if ok, err := roleSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return ds.storage.GetRole(name)
}

func (ds *dataStoreImpl) GetAllRoles(ctx context.Context) ([]*storage.Role, error) {
	if ok, err := roleSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return ds.storage.GetAllRoles()
}

func (ds *dataStoreImpl) AddRole(ctx context.Context, role *storage.Role) error {
	if ok, err := roleSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return ds.storage.AddRole(role)
}

func (ds *dataStoreImpl) UpdateRole(ctx context.Context, role *storage.Role) error {
	if ok, err := roleSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return ds.storage.UpdateRole(role)
}

func (ds *dataStoreImpl) RemoveRole(ctx context.Context, name string) error {
	if ok, err := roleSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return ds.storage.RemoveRole(name)
}
