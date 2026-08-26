package datastore

import (
	"context"

	"github.com/stackrox/rox/central/aiintegration/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	integrationSAC = sac.ForResource(resources.Integration)
)

type dataStore struct {
	store store.Store
}

func (d *dataStore) Get(ctx context.Context, id string) (*storage.AiIntegration, bool, error) {
	if ok, err := integrationSAC.ReadAllowed(ctx); err != nil {
		return nil, false, err
	} else if !ok {
		return nil, false, nil
	}
	return d.store.Get(ctx, id)
}

func (d *dataStore) GetAll(ctx context.Context) ([]*storage.AiIntegration, error) {
	if ok, err := integrationSAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, err
	}

	var integrations []*storage.AiIntegration
	walkFn := func() error {
		integrations = integrations[:0]
		return d.store.Walk(ctx, func(obj *storage.AiIntegration) error {
			integrations = append(integrations, obj)
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(ctx, walkFn); err != nil {
		return nil, err
	}
	return integrations, nil
}

func (d *dataStore) Upsert(ctx context.Context, integration *storage.AiIntegration) error {
	if ok, err := integrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	return d.store.Upsert(ctx, integration)
}

func (d *dataStore) Delete(ctx context.Context, id string) error {
	if ok, err := integrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	return d.store.Delete(ctx, id)
}

func (d *dataStore) Exists(ctx context.Context, id string) (bool, error) {
	if ok, err := integrationSAC.ReadAllowed(ctx); err != nil || !ok {
		return false, err
	}
	return d.store.Exists(ctx, id)
}
