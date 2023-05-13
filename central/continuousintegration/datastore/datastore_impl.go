package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/continuousintegration/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	integrationSAC = sac.ForResource(resources.Integration)

	_ DataStore = (*dataStoreImpl)(nil)
)

type dataStoreImpl struct {
	store store.ContinuousIntegrationStore

	lock sync.RWMutex
}

func (d *dataStoreImpl) GetContinuousIntegrationConfig(ctx context.Context, id string) (*storage.ContinuousIntegrationConfig, bool, error) {
	if err := sac.VerifyAuthzOK(integrationSAC.ReadAllowed(ctx)); err != nil {
		return nil, false, err
	}

	return d.getContinuousIntegrationConfigByID(ctx, id)
}

func (d *dataStoreImpl) GetAllContinuousIntegrationConfigs(ctx context.Context) ([]*storage.ContinuousIntegrationConfig, error) {
	if err := sac.VerifyAuthzOK(integrationSAC.ReadAllowed(ctx)); err != nil {
		return nil, err
	}

	return d.GetAllContinuousIntegrationConfigs(ctx)
}

func (d *dataStoreImpl) AddContinuousIntegrationConfig(ctx context.Context, config *storage.ContinuousIntegrationConfig) (*storage.ContinuousIntegrationConfig, error) {
	if err := sac.VerifyAuthzOK(integrationSAC.WriteAllowed(ctx)); err != nil {
		return nil, err
	}

	if config.GetId() != "" {
		return nil, errox.InvalidArgs.Newf("id should be empty but was %q", config.GetId())
	}
	config.Id = uuid.NewV4().String()

	if err := validateContinuousIntegrationConfig(config); err != nil {
		return nil, err
	}

	d.lock.Lock()
	d.lock.Unlock()
	if err := d.verifyIDDoesNotExist(ctx, config.GetId()); err != nil {
		return nil, err
	}

	if err := d.store.Upsert(ctx, config); err != nil {
		return nil, err
	}
	return config, nil
}

func (d *dataStoreImpl) UpdateContinuousIntegrationConfig(ctx context.Context, config *storage.ContinuousIntegrationConfig) error {
	if err := sac.VerifyAuthzOK(integrationSAC.WriteAllowed(ctx)); err != nil {
		return err
	}

	if err := validateContinuousIntegrationConfig(config); err != nil {
		return err
	}

	d.lock.Lock()
	d.lock.Unlock()

	return d.store.Upsert(ctx, config)
}

func (d *dataStoreImpl) RemoveContinuousIntegrationConfig(ctx context.Context, id string) error {
	if err := sac.VerifyAuthzOK(integrationSAC.WriteAllowed(ctx)); err != nil {
		return err
	}

	return d.store.Delete(ctx, id)
}

func (d *dataStoreImpl) getContinuousIntegrationConfigByID(ctx context.Context, id string) (*storage.ContinuousIntegrationConfig, bool, error) {
	config, found, err := d.store.Get(ctx, id)
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, false, errox.NotFound.Newf("continuous integration config id=%s does not exist", id)
	}
	return config, found, nil
}

func (d *dataStoreImpl) verifyIDDoesNotExist(ctx context.Context, id string) error {
	_, found, err := d.getContinuousIntegrationConfigByID(ctx, id)
	if found {
		return errox.AlreadyExists.Newf("continuous integration config with id %q already exists", id)
	}
	if !errors.Is(err, errox.NotFound) {
		return err
	}
	return nil
}

func validateContinuousIntegrationConfig(config *storage.ContinuousIntegrationConfig) error {
	if config.GetType() == storage.ContinuousIntegrationType_UNSUPPORTED_CONTINUOUS_INTEGRATION {
		return errox.InvalidArgs.Newf("type must be set to %s",
			storage.ContinuousIntegrationType_GITHUB_ACTIONS.String())
	}

	if config.GetId() == "" {
		return errox.InvalidArgs.Newf("id must be set")
	}

	// TODO(dhaus): Verify role mappings.
	return nil
}
