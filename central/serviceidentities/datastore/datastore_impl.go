package datastore

import (
	"context"

	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/central/serviceidentities/internal/store"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/sac"
)

var (
	serviceIdentitiesSAC = sac.ForResource(resources.ServiceIdentity)
)

type dataStoreImpl struct {
	storage store.Store
}

func (ds *dataStoreImpl) GetServiceIdentities(ctx context.Context) ([]*storage.ServiceIdentity, error) {
	if ok, err := serviceIdentitiesSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return ds.storage.GetAll(ctx)
}

func (ds *dataStoreImpl) AddServiceIdentity(ctx context.Context, identity *storage.ServiceIdentity) error {
	if ok, err := serviceIdentitiesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return ds.storage.Upsert(ctx, identity)
}
