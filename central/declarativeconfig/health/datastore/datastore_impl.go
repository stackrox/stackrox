package datastore

import (
	"context"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/declarativeconfig/health/datastore/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
)

type datastoreImpl struct {
	store store.Store
}

func (ds *datastoreImpl) GetDeclarativeConfigs(ctx context.Context) ([]*storage.DeclarativeConfigHealth, error) {
	var declarativeConfigHealths []*storage.DeclarativeConfigHealth
	walkFn := func() error {
		declarativeConfigHealths = declarativeConfigHealths[:0]
		return ds.store.Walk(ctx, func(obj *storage.DeclarativeConfigHealth) error {
			declarativeConfigHealths = append(declarativeConfigHealths, obj)

			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(walkFn); err != nil {
		return nil, err
	}
	return declarativeConfigHealths, nil
}

func (ds *datastoreImpl) UpsertDeclarativeConfig(ctx context.Context, configHealth *storage.DeclarativeConfigHealth) error {
	return ds.store.Upsert(ctx, configHealth)
}

func (ds *datastoreImpl) RemoveDeclarativeConfig(ctx context.Context, id string) error {
	_, exists, err := ds.GetDeclarativeConfig(ctx, id)
	if err != nil {
		return errors.Errorf("failed to retrieve config health %q", id)
	}
	if !exists {
		return errox.NotFound.Newf("unable to find config health for declarative config %q", id)
	}

	return ds.store.Delete(ctx, id)
}

func (ds *datastoreImpl) GetDeclarativeConfig(ctx context.Context, id string) (*storage.DeclarativeConfigHealth, bool, error) {
	return ds.store.Get(ctx, id)
}

func (ds *datastoreImpl) UpdateErrorMessageForDeclarativeConfig(ctx context.Context, id string, errToUpdate error) error {
	existingHealth, exists, err := ds.GetDeclarativeConfig(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return errox.NotFound.Newf("unable to find config health for declarative config %q", id)
	}

	existingHealth.ErrorMessage = errToUpdate.Error()
	existingHealth.LastTimestamp = timestamp.TimestampNow()
	existingHealth.Status = storage.DeclarativeConfigHealth_UNHEALTHY

	return ds.UpsertDeclarativeConfig(ctx, existingHealth)
}
