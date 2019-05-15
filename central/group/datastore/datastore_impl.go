package datastore

import (
	"context"
	"errors"

	"github.com/stackrox/rox/central/group/datastore/internal/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	groupSAC = sac.ForResource(resources.Group)
)

type dataStoreImpl struct {
	storage store.Store
}

func (ds *dataStoreImpl) Get(ctx context.Context, props *storage.GroupProperties) (*storage.Group, error) {
	if ok, err := groupSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return ds.storage.Get(props)
}

func (ds *dataStoreImpl) GetAll(ctx context.Context) ([]*storage.Group, error) {
	if ok, err := groupSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return ds.storage.GetAll()
}

func (ds *dataStoreImpl) Walk(ctx context.Context, authProviderID string, attributes map[string][]string) ([]*storage.Group, error) {
	if ok, err := groupSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return ds.storage.Walk(authProviderID, attributes)
}

func (ds *dataStoreImpl) Add(ctx context.Context, group *storage.Group) error {
	if ok, err := groupSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return ds.storage.Add(group)
}

func (ds *dataStoreImpl) Update(ctx context.Context, group *storage.Group) error {
	if ok, err := groupSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return ds.storage.Update(group)
}

func (ds *dataStoreImpl) Upsert(ctx context.Context, group *storage.Group) error {
	if ok, err := groupSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return ds.storage.Upsert(group)
}

func (ds *dataStoreImpl) Mutate(ctx context.Context, remove, update, add []*storage.Group) error {
	if ok, err := groupSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return ds.storage.Mutate(remove, update, add)
}

func (ds *dataStoreImpl) Remove(ctx context.Context, props *storage.GroupProperties) error {
	if ok, err := groupSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return ds.storage.Remove(props)
}
