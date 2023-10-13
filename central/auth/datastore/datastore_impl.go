package datastore

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/auth/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	_ DataStore = (*datastoreImpl)(nil)
)

type datastoreImpl struct {
	store store.Store
}

func (d *datastoreImpl) GetAuthM2MConfig(ctx context.Context, id string) (*storage.AuthMachineToMachineConfig, bool, error) {
	return d.store.Get(ctx, id)
}

func (d *datastoreImpl) ListAuthM2MConfigs(ctx context.Context) ([]*storage.AuthMachineToMachineConfig, error) {
	return d.store.GetAll(ctx)
}

func (d *datastoreImpl) AddAuthM2MConfig(ctx context.Context, config *storage.AuthMachineToMachineConfig) (*storage.AuthMachineToMachineConfig, error) {
	if config == nil {
		return nil, errox.InvalidArgs.New("empty config given")
	}
	config.Id = uuid.NewV4().String()
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	if err := d.store.Upsert(ctx, config); err != nil {
		return nil, err
	}
	return config, nil
}

func (d *datastoreImpl) UpdateAuthM2MConfig(ctx context.Context, config *storage.AuthMachineToMachineConfig) (*storage.AuthMachineToMachineConfig, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	if err := d.store.Upsert(ctx, config); err != nil {
		return nil, err
	}

	return config, nil
}

func (d *datastoreImpl) RemoveAuthM2MConfig(ctx context.Context, id string) error {
	return d.store.Delete(ctx, id)
}

func validateConfig(config *storage.AuthMachineToMachineConfig) error {
	if config.GetId() == "" {
		return errox.InvalidArgs.New("empty ID given")
	}

	duration, err := time.ParseDuration(config.GetTokenExpirationDuration())
	if err != nil {
		return errox.InvalidArgs.New("invalid token expiration duration given").CausedBy(err)
	}

	if duration < time.Minute || duration > 24*time.Hour {
		return errox.InvalidArgs.Newf("token expiration must be between 1 minute and 24 hours, but was %s",
			duration.String())
	}

	if config.GetType() == storage.AuthMachineToMachineConfig_GENERIC && config.GetGeneric().GetIssuer() == "" {
		return errox.InvalidArgs.Newf("type %s was used, but no configuration for the issuer was given",
			storage.AuthMachineToMachineConfig_GENERIC)
	}

	return nil
}
