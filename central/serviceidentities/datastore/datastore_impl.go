package datastore

import (
	"context"

	"github.com/stackrox/rox/central/serviceidentities/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	administrationSAC = sac.ForResource(resources.Administration)
)

type dataStoreImpl struct {
	storage store.Store
}

func (ds *dataStoreImpl) ProcessServiceIdentities(ctx context.Context, fn func(obj *storage.ServiceIdentity) error) error {
	if ok, err := administrationSAC.ReadAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return nil
	}

	return ds.storage.Walk(ctx, fn)
}

func (ds *dataStoreImpl) AddServiceIdentity(ctx context.Context, identity *storage.ServiceIdentity) error {
	if ok, err := administrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return ds.storage.Upsert(ctx, identity)
}
