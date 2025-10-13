package datastore

import (
	"context"
	"testing"

	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/rules/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
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

	// SearchRules returns the rules for the given query
	SearchRules(ctx context.Context, query *v1.Query) ([]*storage.ComplianceOperatorRuleV2, error)

	// DeleteRulesByCluster removes rule by cluster id
	DeleteRulesByCluster(ctx context.Context, clusterID string) error

	// GetControlsByRulesAndBenchmarks returns controls by a list of rule names group by control, standard and rule name.
	GetControlsByRulesAndBenchmarks(ctx context.Context, ruleNames []string, benchmarkNames []string) ([]*ControlResult, error)
}

// New returns an instance of DataStore.
func New(complianceRuleStorage pgStore.Store, db postgres.DB) DataStore {
	return &datastoreImpl{
		store: complianceRuleStorage,
		db:    db,
	}
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) DataStore {
	store := pgStore.New(pool)
	return New(store, pool)
}
