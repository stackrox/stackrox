package datastore

import (
	"context"

	"github.com/stackrox/rox/central/externalbackups/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	integrationSAC = sac.ForResource(resources.Integration)
)

type datastoreImpl struct {
	store store.Store
}

func (ds *datastoreImpl) ForEachBackup(ctx context.Context, fn func(obj *storage.ExternalBackup) error) error {
	if ok, err := integrationSAC.ReadAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return nil
	}

	return ds.store.Walk(ctx, fn)
}

func (ds *datastoreImpl) GetBackup(ctx context.Context, id string) (*storage.ExternalBackup, bool, error) {
	if ok, err := integrationSAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, false, err
	}

	return ds.store.Get(ctx, id)
}

func (ds *datastoreImpl) UpsertBackup(ctx context.Context, backup *storage.ExternalBackup) error {
	if ok, err := integrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return ds.store.Upsert(ctx, backup)
}

func (ds *datastoreImpl) RemoveBackup(ctx context.Context, id string) error {
	if ok, err := integrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return ds.store.Delete(ctx, id)
}
