package datastore

import (
	"context"
	"testing"

	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/remediations/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
)

// DataStore is the entry point for storing/retrieving compliance operator scan objects.
//
//go:generate mockgen-wrapper
type DataStore interface {
	// GetRemediation retrieves the remediation object from the database
	GetRemediation(ctx context.Context, id string) (*storage.ComplianceOperatorRemediationV2, bool, error)

	// UpsertRemediation adds the remediation object to the database
	UpsertRemediation(ctx context.Context, result *storage.ComplianceOperatorRemediationV2) error

	// DeleteRemediation removes a remediation object from the database
	DeleteRemediation(ctx context.Context, id string) error

	// GetRemediationByCluster retrieves remediation objects by cluster
	GetRemediationsByCluster(ctx context.Context, clusterID string) ([]*storage.ComplianceOperatorRemediationV2, error)

	// DeleteRemediationByCluster deletes a remediation by cluster
	DeleteRemediationsByCluster(ctx context.Context, clusterID string) error

	// SearchRemediations returns the remediations for the given query
	SearchRemediations(ctx context.Context, query *v1.Query) ([]*storage.ComplianceOperatorRemediationV2, error)
}

// New returns an instance of DataStore.
func New(remediationStorage pgStore.Store) DataStore {
	return &datastoreImpl{
		store: remediationStorage,
	}
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) DataStore {
	store := pgStore.New(pool)
	return New(store)
}
