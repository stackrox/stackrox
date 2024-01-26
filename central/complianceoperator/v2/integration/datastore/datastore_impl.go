package datastore

import (
	"context"

	"github.com/pkg/errors"
	integrationsSearch "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore/search"
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
	complianceSAC = sac.ForResource(resources.Compliance)
)

type datastoreImpl struct {
	storage  postgres.Store
	searcher integrationsSearch.Searcher
}

// GetComplianceIntegration is pass-through to the underlying store.
func (ds *datastoreImpl) GetComplianceIntegration(ctx context.Context, id string) (*storage.ComplianceIntegration, bool, error) {
	return ds.storage.Get(ctx, id)
}

// GetComplianceIntegrationByCluster provides an in memory layer on top of the underlying DB based storage.
func (ds *datastoreImpl) GetComplianceIntegrationByCluster(ctx context.Context, clusterID string) ([]*storage.ComplianceIntegration, error) {
	return ds.storage.GetByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, clusterID).ProtoQuery())
}

// GetComplianceIntegrations provides an in memory layer on top of the underlying DB based storage.
func (ds *datastoreImpl) GetComplianceIntegrations(ctx context.Context, query *v1.Query) ([]*storage.ComplianceIntegration, error) {
	return ds.storage.GetByQuery(ctx, query)
}

// AddComplianceIntegration is pass-through to the underlying store.
func (ds *datastoreImpl) AddComplianceIntegration(ctx context.Context, integration *storage.ComplianceIntegration) (string, error) {
	if ok, err := complianceSAC.WriteAllowed(ctx); err != nil {
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
	if ok, err := complianceSAC.WriteAllowed(ctx); err != nil {
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
	if ok, err := complianceSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	return ds.storage.Delete(ctx, id)
}

// RemoveComplianceIntegrationByCluster removes all the compliance integrations for a cluster
func (ds *datastoreImpl) RemoveComplianceIntegrationByCluster(ctx context.Context, clusterID string) error {
	if ok, err := complianceSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	_, storeErr := ds.storage.DeleteByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, clusterID).ProtoQuery())
	return storeErr
}

// CountIntegrations returns count of integrations matching query
func (d *datastoreImpl) CountIntegrations(ctx context.Context, q *v1.Query) (int, error) {
	return d.searcher.Count(ctx, q)
}
