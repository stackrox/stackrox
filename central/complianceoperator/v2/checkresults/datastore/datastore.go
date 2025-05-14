package datastore

import (
	"context"
	"testing"

	checkResultSearch "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore/search"
	checkresultsSearch "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore/search"
	store "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/search"
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

	// ComplianceProfileResultStats retrieves the profile result stats specified by query
	ComplianceProfileResultStats(ctx context.Context, query *v1.Query) ([]*ResourceResultCountByProfile, error)

	// ComplianceProfileResults retrieves the profile results specified by query
	ComplianceProfileResults(ctx context.Context, query *v1.Query) ([]*ResourceResultsByProfile, error)

	// CountCheckResults returns number of scan results specified by query
	CountCheckResults(ctx context.Context, q *v1.Query) (int, error)

	// DeleteResultsByCluster scan results associated with cluster
	DeleteResultsByCluster(ctx context.Context, clusterID string) error

	// DeleteResultsByScanConfigAndCluster deletes scan results associated with scan config and cluster
	DeleteResultsByScanConfigAndCluster(ctx context.Context, scanConfigName string, clusterIDs []string) error

	// DeleteResultsByScanConfigAndRules deletes scan results associated with scan config and rules reference IDs
	DeleteResultsByScanConfigAndRules(ctx context.Context, scanConfigName string, ruleRefIds []string) error

	// GetComplianceCheckResult returns the instance of the result specified by ID
	GetComplianceCheckResult(ctx context.Context, complianceResultID string) (*storage.ComplianceOperatorCheckResultV2, bool, error)

	// CountByField retrieves the distinct scan result counts specified by query based on specified search field
	CountByField(ctx context.Context, query *v1.Query, field search.FieldLabel) (int, error)

	// WalkByQuery gets one row at a time and applies function per row
	WalkByQuery(ctx context.Context, query *v1.Query, fn func(deployment *storage.ComplianceOperatorCheckResultV2) error) error
}

// New returns the datastore wrapper for compliance operator check results
func New(store store.Store, db postgres.DB, searcher checkResultSearch.Searcher) DataStore {
	return &datastoreImpl{store: store, db: db, searcher: searcher}
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) DataStore {
	store := store.New(pool)
	searcher := checkresultsSearch.New(store)
	return New(store, pool, searcher)
}
