package datastore

import (
	"context"
	"testing"

	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/rules/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
)

// DataStore is the entry point for storing/retrieving compliance operator rules.
//
//go:generate mockgen-wrapper
type DataStore interface {
	// UpsertRule adds the rule to the database
	UpsertRule(ctx context.Context, rule *storage.ComplianceOperatorRuleV2) error

	// DeleteRule removes a rule from the database
	DeleteRule(ctx context.Context, id string) error

	// GetRulesByCluster retrieves rules by cluster
	GetRulesByCluster(ctx context.Context, clusterID string) ([]*storage.ComplianceOperatorRuleV2, error)
}

// New returns an instance of DataStore.
func New(complianceRuleStorage pgStore.Store) DataStore {
	return &datastoreImpl{
		store: complianceRuleStorage,
	}
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool postgres.DB) DataStore {
	store := pgStore.New(pool)
	return New(store)
}
