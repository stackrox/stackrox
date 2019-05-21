package datastore

import (
	"context"

	"github.com/pkg/errors"
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

	return ds.store.ListBackups()
}

func (ds *datastoreImpl) GetBackup(ctx context.Context, id string) (*storage.ExternalBackup, error) {
	if ok, err := externalBkpSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return ds.store.GetBackup(id)
}

func (ds *datastoreImpl) UpsertBackup(ctx context.Context, backup *storage.ExternalBackup) error {
	if ok, err := externalBkpSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return ds.store.UpsertBackup(backup)
}

func (ds *datastoreImpl) RemoveBackup(ctx context.Context, id string) error {
	if ok, err := externalBkpSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return ds.store.RemoveBackup(id)
}
