package datastore

import (
	"context"

	"github.com/stackrox/rox/central/imageintegration/index"
	"github.com/stackrox/rox/central/imageintegration/search"
	"github.com/stackrox/rox/central/imageintegration/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	imageIntegrationSAC = sac.ForResource(resources.ImageIntegration)
)

type datastoreImpl struct {
	storage           store.Store
	indexer           index.Indexer
	formattedSearcher search.Searcher
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return ds.formattedSearcher.Search(ctx, q)
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
	err := ds.storage.Upsert(ctx, integration)
	if err != nil {
		return "", err
	}
	return integration.Id, ds.indexer.AddImageIntegration(integration)
}

// UpdateImageIntegration is pass-through to the underlying store.
func (ds *datastoreImpl) UpdateImageIntegration(ctx context.Context, integration *storage.ImageIntegration) error {
	if ok, err := imageIntegrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	err := ds.storage.Upsert(ctx, integration)
	if err != nil {
		return err
	}
	return ds.indexer.AddImageIntegration(integration)
}

// RemoveImageIntegration is pass-through to the underlying store.
func (ds *datastoreImpl) RemoveImageIntegration(ctx context.Context, id string) error {
	if ok, err := imageIntegrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	if err := ds.storage.Delete(ctx, id); err != nil {
		return err
	}
	return ds.indexer.DeleteImageIntegration(id)
}

func (ds *datastoreImpl) buildIndex(ctx context.Context) error {
	if features.PostgresDatastore.Enabled() {
		return nil
	}
	imageIntegrations, err := ds.storage.GetAll(ctx)
	log.Infof("[STARTUP] Found %d Image Integrations to be indexed", len(imageIntegrations))
	if err != nil {
		return err
	}
	err = ds.indexer.AddImageIntegrations(imageIntegrations)
	if err != nil {
		return err
	}
	log.Infof("[STARTUP] Successfully indexed %d Image Integrations", len(imageIntegrations))
	return nil
}

// SearchImageIntegrations
func (ds *datastoreImpl) SearchImageIntegrations(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return ds.formattedSearcher.SearchImageIntegrations(ctx, q)
}
