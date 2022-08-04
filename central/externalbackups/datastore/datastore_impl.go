package datastore

import (
	"context"

	"github.com/stackrox/rox/central/externalbackups/internal/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	externalBkpSAC = sac.ForResource(resources.BackupPlugins)
)

type datastoreImpl struct {
	store store.Store
}

func (ds *datastoreImpl) ListBackups(ctx context.Context) ([]*storage.ExternalBackup, error) {
	if ok, err := externalBkpSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return ds.store.GetAll(ctx)
}

func (ds *datastoreImpl) GetBackup(ctx context.Context, id string) (*storage.ExternalBackup, bool, error) {
	if ok, err := externalBkpSAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, false, err
	}

	return ds.store.Get(ctx, id)
}

func (ds *datastoreImpl) UpsertBackup(ctx context.Context, backup *storage.ExternalBackup) error {
	if ok, err := externalBkpSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return ds.store.Upsert(ctx, backup)
}

func (ds *datastoreImpl) RemoveBackup(ctx context.Context, id string) error {
	if ok, err := externalBkpSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return ds.store.Delete(ctx, id)
}
