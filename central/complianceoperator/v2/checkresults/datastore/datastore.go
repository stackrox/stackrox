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

	// UpsertReceivedResult upserts a result just received from sensor and, atomically with the
	// write, derives its data_freshness instead of trusting the caller's value: "received" does
	// not always mean "new" (sensor resends every existing check result on reconnect regardless of
	// whether anything changed, and a misbehaving compliance operator could republish a result
	// without a real rescan behind it). A result is CURRENT when it has no prior stored row and a
	// non-nil LastStartedTime (a nil LastStartedTime cannot be evaluated and stays UNKNOWN, matching
	// MarkStalePriorResultsByScanRef's treatment of nil timestamps), or when its LastStartedTime is
	// newer than what is currently stored; otherwise the existing row's freshness AND LastStartedTime
	// are both carried over unchanged (a replay must not regress the stored timestamp, which would
	// corrupt later staleness-by-age decisions). The read and the write happen under the same
	// KeyFence used by MarkStalePriorResultsByScanRef, so an ingest decision can't be based on a
	// stale read that a concurrent timeout marking (or another ingest) has since invalidated, and
	// cannot revert that concurrent write. On a lookup error, no write happens: an error here must
	// not silently replace a known freshness value with UNKNOWN.
	UpsertReceivedResult(ctx context.Context, result *storage.ComplianceOperatorCheckResultV2) error

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

	// MarkStalePriorResultsByScanRef marks STALE the check results of the given scan (identified
	// by its scan ref ID) that predate olderThan, leaving the rest unchanged. It is the
	// write-time staleness hook for a timed-out scan: the results the timed-out run did not
	// refresh (leftovers from a previous run, with an older last started time) are known-stale
	// immediately. Results the run refreshed carry its start time (olderThan) and stay as they
	// are. Results with a nil last started time are left unchanged (freshness cannot be
	// evaluated). Only rows whose stored value changes are written. Returns the number of rows
	// updated.
	MarkStalePriorResultsByScanRef(ctx context.Context, scanRefID string, olderThan time.Time) (int64, error)
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
