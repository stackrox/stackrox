package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore/search"
	checkresultsSearch "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore/search"
	store "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
)

// DataStore defines the possible interactions with compliance operator check results
//
//go:generate mockgen-wrapper
type DataStore interface {
	// UpsertResult adds the result to the database
	UpsertResult(ctx context.Context, result *storage.ComplianceOperatorCheckResultV2) error

	// DeleteResult removes a result from the database
	DeleteResult(ctx context.Context, id string) error

	// SearchComplianceCheckResults retrieves the scan results specified by query
	SearchComplianceCheckResults(ctx context.Context, query *v1.Query) ([]*storage.ComplianceOperatorCheckResultV2, error)

	// ComplianceCheckResultStats retrieves the scan results stats specified by query for the scan configuration
	ComplianceCheckResultStats(ctx context.Context, query *v1.Query) ([]*ResourceResultCountByClusterScan, error)

	// ComplianceClusterStats retrieves the scan result stats specified by query for the clusters
	ComplianceClusterStats(ctx context.Context, query *v1.Query) ([]*ResultStatusCountByCluster, error)

	// ComplianceProfileResultStats retrieves the profile results stats specified by query for the scan configuration
	ComplianceProfileResultStats(ctx context.Context, query *v1.Query) ([]*ResourceResultCountByProfile, error)

	// ComplianceClusterStatsCount retrieves the distinct scan result counts specified by query for the clusters
	ComplianceClusterStatsCount(ctx context.Context, query *v1.Query) (int, error)

	// CountCheckResults returns number of scan results specified by query
	CountCheckResults(ctx context.Context, q *v1.Query) (int, error)

	// Delete scan results associated with cluster
	DeleteResultsByCluster(ctx context.Context, clusterID string) error

	// GetComplianceCheckResult returns the instance of the result specified by ID
	GetComplianceCheckResult(ctx context.Context, complianceResultID string) (*storage.ComplianceOperatorCheckResultV2, bool, error)
}

// New returns the datastore wrapper for compliance operator check results
func New(store store.Store, db postgres.DB, searcher search.Searcher) DataStore {
	return &datastoreImpl{store: store, db: db, searcher: searcher}
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) DataStore {
	store := store.New(pool)
	searcher := checkresultsSearch.New(store)
	return New(store, pool, searcher)
}
