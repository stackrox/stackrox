package datastore

import (
	"context"

	"github.com/stackrox/rox/central/sensorupgradeconfig/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sac"
)

type dataStore struct {
	store store.Store

	autoTrigger concurrency.Flag
}

var (
	sacHelper = sac.ForResource(resources.SensorUpgradeConfig)
)

func (d *dataStore) initialize() error {
	cfg, err := d.store.GetSensorUpgradeConfig()
	if err != nil {
		return err
	}
	d.autoTrigger.Set(cfg.GetEnableAutoUpgrade())
	return nil
}

func (d *dataStore) GetSensorUpgradeConfig(ctx context.Context) (*storage.SensorUpgradeConfig, error) {
	if ok, err := sacHelper.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}
	return d.store.GetSensorUpgradeConfig()
}

func (d *dataStore) UpsertSensorUpgradeConfig(ctx context.Context, sensorUpgradeConfig *storage.SensorUpgradeConfig) error {
	if ok, err := sacHelper.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if err := d.store.UpsertSensorUpgradeConfig(sensorUpgradeConfig); err != nil {
		return err
	}
	d.autoTrigger.Set(sensorUpgradeConfig.GetEnableAutoUpgrade())
	return nil
}

func (d *dataStore) AutoTriggerSetting() *concurrency.Flag {
	return &d.autoTrigger
}
