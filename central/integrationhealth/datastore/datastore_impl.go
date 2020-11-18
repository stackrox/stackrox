package datastore

import (
	"context"

	"github.com/stackrox/rox/central/integrationhealth/store/rocksdb"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	imageSAC    = sac.ForResource(resources.ImageIntegration)
	notifierSAC = sac.ForResource(resources.Notifier)
	backupSAC   = sac.ForResource(resources.BackupPlugins)
)

type datastoreImpl struct {
	store rocksdb.Store
}

func (ds *datastoreImpl) GetRegistriesAndScanners(ctx context.Context) ([]*storage.IntegrationHealth, error) {
	if ok, err := imageSAC.ReadAllowed(ctx); err != nil || !ok {
		return nil,
			status.Errorf(codes.Internal, "Failed to retrieve health for registries and scanners: %v", err)
	}
	return ds.getIntegrationsOfType(storage.IntegrationHealth_IMAGE_INTEGRATION)

}

func (ds *datastoreImpl) GetNotifierPlugins(ctx context.Context) ([]*storage.IntegrationHealth, error) {
	if ok, err := notifierSAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, status.Errorf(codes.Internal, "Failed to retrieve health for notifiers: %v", err)
	}
	return ds.getIntegrationsOfType(storage.IntegrationHealth_NOTIFIER)
}

func (ds *datastoreImpl) GetBackupPlugins(ctx context.Context) ([]*storage.IntegrationHealth, error) {
	if ok, err := backupSAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, status.Errorf(codes.Internal, "Failed to retrieve health for backup plugins: %v", err)
	}
	return ds.getIntegrationsOfType(storage.IntegrationHealth_BACKUP)
}

func (ds *datastoreImpl) UpdateIntegrationHealth(ctx context.Context, integrationHealth *storage.IntegrationHealth) error {
	if err := writeAllowed(ctx); err != nil {
		return status.Errorf(codes.Internal, "Failed to update health for integration %s: %v",
			integrationHealth.Id, err)
	}
	return ds.store.Upsert(integrationHealth)
}

func (ds *datastoreImpl) RemoveIntegrationHealth(ctx context.Context, id string) error {
	if err := writeAllowed(ctx); err != nil {
		return status.Errorf(codes.Internal, "Failed to remove health for integration %s: %v", id, err)
	}
	return ds.store.Delete(id)
}

func writeAllowed(ctx context.Context) error {
	if ok, err := imageSAC.WriteAllowed(ctx); err != nil || !ok {
		return status.Error(codes.Internal, "Permission denied")
	}
	if ok, err := notifierSAC.WriteAllowed(ctx); err != nil || !ok {
		return status.Error(codes.Internal, "Permission denied")
	}
	if ok, err := backupSAC.WriteAllowed(ctx); err != nil || !ok {
		return status.Error(codes.Internal, "Permission denied")
	}
	return nil
}

func (ds *datastoreImpl) getIntegrationsOfType(integrationType storage.IntegrationHealth_Type) ([]*storage.IntegrationHealth, error) {
	var integrationHealth []*storage.IntegrationHealth
	err := ds.store.Walk(func(obj *storage.IntegrationHealth) error {
		if obj.GetType() == integrationType {
			integrationHealth = append(integrationHealth, obj)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return integrationHealth, nil
}
