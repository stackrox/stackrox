package datastore

import (
	"context"
	"testing"

	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/integration/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
)

// DataStore is the entry point for storing/retrieving compliance operator metadata.
//
//go:generate mockgen-wrapper
type DataStore interface {
	// GetComplianceIntegration retrieves the compliance integration from the store
	GetComplianceIntegration(ctx context.Context, id string) (*storage.ComplianceIntegration, bool, error)
	// GetComplianceIntegrationByCluster retrieves the compliance integrations from the store by cluster id
	GetComplianceIntegrationByCluster(ctx context.Context, clusterID string) ([]*storage.ComplianceIntegration, error)
	// GetComplianceIntegrations retrieves all the compliance integrations from the store
	GetComplianceIntegrations(ctx context.Context, query *v1.Query) ([]*storage.ComplianceIntegration, error)
	// GetComplianceIntegrationsView provides an in memory layer on top of the underlying DB based storage.
	GetComplianceIntegrationsView(ctx context.Context, query *v1.Query) ([]*IntegrationDetails, error)
	// AddComplianceIntegration adds a compliance integration to the store
	AddComplianceIntegration(ctx context.Context, integration *storage.ComplianceIntegration) (string, error)
	// UpdateComplianceIntegration updates a compliance integration to the store
	UpdateComplianceIntegration(ctx context.Context, integration *storage.ComplianceIntegration) error
	// RemoveComplianceIntegration removes the compliance integration from the store
	RemoveComplianceIntegration(ctx context.Context, id string) error
	// RemoveComplianceIntegrationByCluster removes all the compliance integrations for a cluster
	RemoveComplianceIntegrationByCluster(ctx context.Context, clusterID string) error
	// CountIntegrations returns count of integrations matching query
	CountIntegrations(ctx context.Context, q *v1.Query) (int, error)
}

// New returns an instance of DataStore.
func New(complianceIntegrationStorage pgStore.Store, pool postgres.DB) DataStore {
	ds := &datastoreImpl{
		storage: complianceIntegrationStorage,
		db:      pool,
	}
	return ds
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) DataStore {
	store := pgStore.New(pool)
	return New(store, pool)
}
