package datastore

import (
	"context"

	"github.com/pkg/errors"
	store "github.com/stackrox/rox/central/complianceoperator/v2/integration/store/postgres"
	complianceUtils "github.com/stackrox/rox/central/complianceoperator/v2/utils"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	complianceSAC = sac.ForResource(resources.Compliance)
)

type datastoreImpl struct {
	storage store.Store
	db      postgres.DB
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

// GetComplianceIntegrationsView provides an in memory layer on top of the underlying DB based storage.
func (ds *datastoreImpl) GetComplianceIntegrationsView(ctx context.Context, query *v1.Query) ([]*IntegrationDetails, error) {
	var err error
	query, err = complianceUtils.WithSACFilter(ctx, resources.Compliance, query)
	if err != nil {
		return nil, err
	}

	cloned := query.CloneVT()
	cloned.SetSelects([]*v1.QuerySelect{
		search.NewQuerySelect(search.Cluster).Proto(),
		search.NewQuerySelect(search.ClusterID).Proto(),
		search.NewQuerySelect(search.ClusterType).Proto(),
		search.NewQuerySelect(search.ClusterPlatformType).Proto(),
		search.NewQuerySelect(search.ComplianceOperatorInstalled).Proto(),
		search.NewQuerySelect(search.ComplianceOperatorVersion).Proto(),
		search.NewQuerySelect(search.ComplianceOperatorStatus).Proto(),
		search.NewQuerySelect(search.ComplianceOperatorIntegrationID).Proto(),
	})

	results, err := pgSearch.RunSelectRequestForSchema[IntegrationDetails](ctx, ds.db, schema.ComplianceIntegrationsSchema, cloned)
	if err != nil {
		return nil, err
	}

	return results, nil
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
	integration.SetId(uuid.NewV4().String())

	err := ds.storage.Upsert(ctx, integration)
	if err != nil {
		return "", err
	}
	return integration.GetId(), nil
}

// UpdateComplianceIntegration is pass-through to the underlying store.
func (ds *datastoreImpl) UpdateComplianceIntegration(ctx context.Context, integration *storage.ComplianceIntegration) error {
	if ok, err := complianceSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if integration.GetId() == "" {
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

	return ds.storage.DeleteByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, clusterID).ProtoQuery())
}

// CountIntegrations returns count of integrations matching query
func (ds *datastoreImpl) CountIntegrations(ctx context.Context, q *v1.Query) (int, error) {
	return ds.storage.Count(ctx, q)
}
