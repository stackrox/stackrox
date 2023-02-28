package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/integrationhealth/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	integrationSAC = sac.ForResource(resources.Integration)
)

type datastoreImpl struct {
	store store.Store
}

func (ds *datastoreImpl) GetRegistriesAndScanners(ctx context.Context) ([]*storage.IntegrationHealth, error) {
	if err := sac.VerifyAuthzOK(integrationSAC.ReadAllowed(ctx)); err != nil {
		return nil, err
	}
	return ds.getIntegrationsOfType(ctx, storage.IntegrationHealth_IMAGE_INTEGRATION)
}

func (ds *datastoreImpl) GetNotifierPlugins(ctx context.Context) ([]*storage.IntegrationHealth, error) {
	if err := sac.VerifyAuthzOK(integrationSAC.ReadAllowed(ctx)); err != nil {
		return nil, err
	}
	return ds.getIntegrationsOfType(ctx, storage.IntegrationHealth_NOTIFIER)
}

func (ds *datastoreImpl) GetBackupPlugins(ctx context.Context) ([]*storage.IntegrationHealth, error) {
	if err := sac.VerifyAuthzOK(integrationSAC.ReadAllowed(ctx)); err != nil {
		return nil, err
	}

	return ds.getIntegrationsOfType(ctx, storage.IntegrationHealth_BACKUP)
}

func (ds *datastoreImpl) GetDeclarativeConfigs(ctx context.Context) ([]*storage.IntegrationHealth, error) {
	if err := sac.VerifyAuthzOK(integrationSAC.ReadAllowed(ctx)); err != nil {
		return nil, err
	}

	return ds.getIntegrationsOfType(ctx, storage.IntegrationHealth_DECLARATIVE_CONFIG)
}

func (ds *datastoreImpl) UpdateIntegrationHealth(ctx context.Context, integrationHealth *storage.IntegrationHealth) error {
	if err := sac.VerifyAuthzOK(integrationSAC.WriteAllowed(ctx)); err != nil {
		return err
	}

	if err := validateIntegrationHealthType(integrationHealth.GetType()); err != nil {
		return err
	}

	return ds.store.Upsert(ctx, integrationHealth)
}

func (ds *datastoreImpl) RemoveIntegrationHealth(ctx context.Context, id string) error {
	if err := sac.VerifyAuthzOK(integrationSAC.WriteAllowed(ctx)); err != nil {
		return err
	}

	_, exists, err := ds.GetIntegrationHealth(ctx, id)
	if err != nil {
		return errors.Errorf("failed to retrieve integration health %q", id)
	}
	if !exists {
		return errox.NotFound.Newf("unable to find integration health for integration %q", id)
	}

	return ds.store.Delete(ctx, id)
}

func (ds *datastoreImpl) GetIntegrationHealth(ctx context.Context, id string) (*storage.IntegrationHealth, bool, error) {
	if err := sac.VerifyAuthzOK(integrationSAC.ReadAllowed(ctx)); err != nil {
		return nil, false, err
	}

	health, found, err := ds.store.Get(ctx, id)
	if !found || err != nil {
		return nil, false, err
	}

	return health, found, err
}

func (ds *datastoreImpl) getIntegrationsOfType(ctx context.Context, integrationType storage.IntegrationHealth_Type) ([]*storage.IntegrationHealth, error) {
	if err := validateIntegrationHealthType(integrationType); err != nil {
		return nil, err
	}

	var integrationHealth []*storage.IntegrationHealth
	walkFn := func() error {
		integrationHealth = integrationHealth[:0]
		return ds.store.Walk(ctx, func(obj *storage.IntegrationHealth) error {
			if obj.GetType() == integrationType {
				integrationHealth = append(integrationHealth, obj)
			}
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(walkFn); err != nil {
		return nil, err
	}
	return integrationHealth, nil
}

func validateIntegrationHealthType(typ storage.IntegrationHealth_Type) error {
	if typ == storage.IntegrationHealth_UNKNOWN {
		return errox.InvalidArgs.Newf("invalid integration health type %s given", typ)
	}
	return nil
}
