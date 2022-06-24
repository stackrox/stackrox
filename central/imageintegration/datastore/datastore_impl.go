package datastore

import (
	"context"

	"github.com/stackrox/rox/central/imageintegration/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	imageIntegrationSAC = sac.ForResource(resources.ImageIntegration)
)

type datastoreImpl struct {
	storage store.Store
}

// GetImageIntegration is pass-through to the underlying store.
func (ds *datastoreImpl) GetImageIntegration(ctx context.Context, id string) (*storage.ImageIntegration, bool, error) {
	if ok, err := imageIntegrationSAC.ReadAllowed(ctx); err != nil {
		return nil, false, err
	} else if !ok {
		return nil, false, nil
	}
	return ds.storage.Get(ctx, id)
}

// GetImageIntegrations provides an in memory layer on top of the underlying DB based storage.
func (ds *datastoreImpl) GetImageIntegrations(ctx context.Context, request *v1.GetImageIntegrationsRequest) ([]*storage.ImageIntegration, error) {
	if ok, err := imageIntegrationSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	integrations, err := ds.storage.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	integrationSlice := integrations[:0]
	for _, integration := range integrations {
		if request.GetCluster() != "" {
			continue
		}
		if request.GetName() != "" && request.GetName() != integration.GetName() {
			continue
		}
		integrationSlice = append(integrationSlice, integration)
	}
	return integrationSlice, nil
}

// AddImageIntegration is pass-through to the underlying store.
func (ds *datastoreImpl) AddImageIntegration(ctx context.Context, integration *storage.ImageIntegration) (string, error) {
	if ok, err := imageIntegrationSAC.WriteAllowed(ctx); err != nil {
		return "", err
	} else if !ok {
		return "", sac.ErrResourceAccessDenied
	}

	integration.Id = uuid.NewV4().String()
	return integration.Id, ds.storage.Upsert(ctx, integration)
}

// UpdateImageIntegration is pass-through to the underlying store.
func (ds *datastoreImpl) UpdateImageIntegration(ctx context.Context, integration *storage.ImageIntegration) error {
	if ok, err := imageIntegrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return ds.storage.Upsert(ctx, integration)
}

// RemoveImageIntegration is pass-through to the underlying store.
func (ds *datastoreImpl) RemoveImageIntegration(ctx context.Context, id string) error {
	if ok, err := imageIntegrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return ds.storage.Delete(ctx, id)
}
