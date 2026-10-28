package datastore

import (
	"context"
	"testing"
	"time"

	store "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/search"
	"google.golang.org/protobuf/types/known/timestamppb"
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

	// DeleteResultsByScans deletes scan results associated with scans
	DeleteResultsByScans(ctx context.Context, scanRefIds []string) error

	// GetComplianceCheckResult returns the instance of the result specified by ID
	GetComplianceCheckResult(ctx context.Context, complianceResultID string) (*storage.ComplianceOperatorCheckResultV2, bool, error)

	// CountByField retrieves the distinct scan result counts specified by query based on specified search field
	CountByField(ctx context.Context, query *v1.Query, field search.FieldLabel) (int, error)

	// WalkByQuery gets one row at a time and applies function per row
	WalkByQuery(ctx context.Context, query *v1.Query, fn func(deployment *storage.ComplianceOperatorCheckResultV2) error) error

	// DeleteOldResults scan results from a previous run
	DeleteOldResults(ctx context.Context, lastStartedTimestamp *timestamppb.Timestamp, scanRefID string, includeCurrent bool) error

	// UpdateFreshnessByScanConfig recomputes data_freshness for all check results of a scan
	// configuration. A result is STALE when its last started time predates staleBefore, CURRENT
	// when it is more recent, and UNKNOWN when it is nil (freshness cannot be evaluated). Only
	// rows whose stored value changes are written. Returns the number of rows updated.
	UpdateFreshnessByScanConfig(ctx context.Context, scanConfigName string, staleBefore time.Time) (int64, error)

	// MarkStaleByScanRef marks every check result of the given scan (identified by its scan
	// ref ID) as STALE. It is the write-time complement to the freshness sweeper: when a scan
	// times out, the expected fresh results never arrived, so the existing results for that
	// scan are known-stale immediately instead of only after the sweeper's grace window. Only
	// rows not already STALE are written. Returns the number of rows updated.
	MarkStaleByScanRef(ctx context.Context, scanRefID string) (int64, error)
}

// New returns the datastore wrapper for compliance operator check results
func New(store store.Store, db postgres.DB) DataStore {
	return &datastoreImpl{store: store, db: db, keyFence: concurrency.NewKeyFence()}
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) DataStore {
	store := store.New(pool)
	return New(store, pool)
}
