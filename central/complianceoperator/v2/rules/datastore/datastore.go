package datastore

import (
	"context"
	"testing"

	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/rules/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	log = logging.LoggerForModule()
)

// DataStore is the entry point for storing/retrieving compliance operator profiles.
//
//go:generate mockgen-wrapper
type DataStore interface {
	// UpsertRule adds the rule to the database
	UpsertRule(ctx context.Context, result *storage.ComplianceOperatorRuleV2) error

	// UpsertRules adds the rules to the database
	UpsertRules(ctx context.Context, result []*storage.ComplianceOperatorRuleV2) error

	// DeleteRule removes a rule from the database
	DeleteRule(ctx context.Context, id string) error
}

// New returns an instance of DataStore.
func New(complianceRuleStorage pgStore.Store) DataStore {
	ds := &datastoreImpl{
		store: complianceRuleStorage,
	}
	return ds
}

// NewForTestOnly returns an instance of DataStore only for tests.
func NewForTestOnly(_ *testing.T, complianceRuleStorage pgStore.Store) DataStore {
	ds := &datastoreImpl{
		store: complianceRuleStorage,
	}
	return ds
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool postgres.DB) (DataStore, error) {
	store := pgStore.New(pool)
	return New(store), nil
}
