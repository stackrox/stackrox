package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/complianceoperator/v2/integration/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	integrationSAC = sac.ForResource(resources.Integration)
)

type datastoreImpl struct {
	storage postgres.Store
}

// GetComplianceIntegration is pass-through to the underlying store.
func (ds *datastoreImpl) GetComplianceIntegration(ctx context.Context, id string) (*storage.ComplianceIntegration, bool, error) {
	if ok, err := integrationSAC.ReadAllowed(ctx); err != nil {
		return nil, false, err
	} else if !ok {
		return nil, false, nil
	}

	return ds.storage.Get(ctx, id)
}

// GetComplianceIntegrationByCluster provides an in memory layer on top of the underlying DB based storage.
func (ds *datastoreImpl) GetComplianceIntegrationByCluster(ctx context.Context, clusterID string) ([]*storage.ComplianceIntegration, error) {
	if ok, err := integrationSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return ds.storage.GetByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, clusterID).ProtoQuery())
}

// GetComplianceIntegrations provides an in memory layer on top of the underlying DB based storage.
func (ds *datastoreImpl) GetComplianceIntegrations(ctx context.Context, query *v1.Query) ([]*storage.ComplianceIntegration, error) {
	if ok, err := integrationSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return ds.storage.GetByQuery(ctx, query)
}

// AddComplianceIntegration is pass-through to the underlying store.
func (ds *datastoreImpl) AddComplianceIntegration(ctx context.Context, integration *storage.ComplianceIntegration) (string, error) {
	if ok, err := integrationSAC.WriteAllowed(ctx); err != nil {
		return "", err
	} else if !ok {
		return "", sac.ErrResourceAccessDenied
	}

	if integration.GetId() != "" {
		return "", errox.InvalidArgs.Newf("id should be empty but %q provided", integration.GetId())
	}
	integration.Id = uuid.NewV4().String()

	err := ds.storage.Upsert(ctx, integration)
	if err != nil {
		return "", err
	}
	return integration.Id, nil
}

// UpdateComplianceIntegration is pass-through to the underlying store.
func (ds *datastoreImpl) UpdateComplianceIntegration(ctx context.Context, integration *storage.ComplianceIntegration) error {
	if ok, err := integrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if integration.Id == "" {
		return errors.New("Unable to update compliance integration without an ID")
	}

	return ds.storage.Upsert(ctx, integration)
}

// RemoveComplianceIntegration is pass-through to the underlying store.
func (ds *datastoreImpl) RemoveComplianceIntegration(ctx context.Context, id string) error {
	if ok, err := integrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	return ds.storage.Delete(ctx, id)
}

// RemoveComplianceIntegrationByCluster removes all the compliance integrations for a cluster
func (ds *datastoreImpl) RemoveComplianceIntegrationByCluster(ctx context.Context, clusterID string) error {
	if ok, err := integrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	_, storeErr := ds.storage.DeleteByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, clusterID).ProtoQuery())
	return storeErr
}
