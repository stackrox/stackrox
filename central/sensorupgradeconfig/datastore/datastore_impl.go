package datastore

import (
	"context"

	"github.com/stackrox/rox/central/sensorupgradeconfig/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

type dataStore struct {
	store store.Store
}

var (
	sacHelper = sac.ForResource(resources.Administration)
)

func (d *dataStore) GetSensorUpgradeConfig(ctx context.Context) (*storage.SensorUpgradeConfig, error) {
	if ok, err := sacHelper.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}
	config, _, err := d.store.Get(ctx)
	return config, err
}

func (d *dataStore) UpsertSensorUpgradeConfig(ctx context.Context, sensorUpgradeConfig *storage.SensorUpgradeConfig) error {
	if ok, err := sacHelper.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if err := d.store.Upsert(ctx, sensorUpgradeConfig); err != nil {
		return err
	}
	return nil
}
