package datastore

import (
	"context"
	"testing"

	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/suites/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
)

// DataStore is the entry point for storing/retrieving compliance operator suite.
//
//go:generate mockgen-wrapper
type DataStore interface {
	// GetSuite returns the suite for the given id
	GetSuite(ctx context.Context, id string) (*storage.ComplianceOperatorSuiteV2, bool, error)

	// GetSuites return the suites matching the query
	GetSuites(ctx context.Context, query *v1.Query) ([]*storage.ComplianceOperatorSuiteV2, error)

	// GetSuitesByCluster retrieve suites by the cluster
	GetSuitesByCluster(ctx context.Context, clusterID string) ([]*storage.ComplianceOperatorSuiteV2, error)

	// UpsertSuite adds the suite to the database
	UpsertSuite(ctx context.Context, suite *storage.ComplianceOperatorSuiteV2) error

	// UpsertSuites adds the suites to the database
	UpsertSuites(ctx context.Context, suites []*storage.ComplianceOperatorSuiteV2) error

	// DeleteSuite removes a suite from the database
	DeleteSuite(ctx context.Context, id string) error

	// DeleteSuitesByCLuster removes a suite from the database
	DeleteSuitesByCluster(ctx context.Context, clusterID string) error
}

// New returns an instance of DataStore.
func New(complianceSuiteStorage pgStore.Store) DataStore {
	ds := &datastoreImpl{
		store: complianceSuiteStorage,
	}
	return ds
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) DataStore {
	store := pgStore.New(pool)
	return New(store)
}
