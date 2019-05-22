package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/serviceidentities/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
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

	return ds.storage.GetServiceIdentities()
}

func (ds *dataStoreImpl) AddServiceIdentity(ctx context.Context, identity *storage.ServiceIdentity) error {
	if ok, err := serviceIdentitiesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return ds.storage.AddServiceIdentity(identity)
}
