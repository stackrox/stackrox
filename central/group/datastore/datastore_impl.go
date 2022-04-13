package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/group/datastore/internal/store"
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/sac"
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

func (ds *dataStoreImpl) GetFiltered(ctx context.Context, filter func(*storage.GroupProperties) bool) ([]*storage.Group, error) {
	if ok, err := groupSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return ds.storage.GetFiltered(filter)
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
		return sac.ErrResourceAccessDenied
	}

	return ds.storage.Add(group)
}

func (ds *dataStoreImpl) Update(ctx context.Context, group *storage.Group) error {
	if ok, err := groupSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return ds.storage.Update(group)
}

func (ds *dataStoreImpl) Upsert(ctx context.Context, group *storage.Group) error {
	if ok, err := groupSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return ds.storage.Upsert(group)
}

func (ds *dataStoreImpl) Mutate(ctx context.Context, remove, update, add []*storage.Group) error {
	if ok, err := groupSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return ds.storage.Mutate(remove, update, add)
}

func (ds *dataStoreImpl) Remove(ctx context.Context, props *storage.GroupProperties) error {
	if ok, err := groupSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return ds.storage.Remove(props)
}

func (ds *dataStoreImpl) RemoveAllWithAuthProviderID(ctx context.Context, authProviderID string) error {
	groups, err := ds.GetFiltered(ctx, func(properties *storage.GroupProperties) bool {
		return authProviderID == properties.GetAuthProviderId()
	})
	if err != nil {
		return errors.Wrap(err, "collecting associated groups")
	}
	return ds.Mutate(ctx, groups, nil, nil)
}
