package datastore

import (
	"context"
	"testing"

	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/report/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
)

// DataStore is the entry point for storing/retrieving compliance operator report snapshot objects.
//
//go:generate mockgen-wrapper
type DataStore interface {
	// GetSnapshot retrieves the report snapshot object from the database
	GetSnapshot(ctx context.Context, id string) (*storage.ComplianceOperatorReportSnapshotV2, bool, error)
	// SearchSnapshots  returns the snapshots for the given query
	SearchSnapshots(ctx context.Context, query *v1.Query) ([]*storage.ComplianceOperatorReportSnapshotV2, error)

	// UpsertSnapshot adds the report snapshot object to the database
	UpsertSnapshot(ctx context.Context, result *storage.ComplianceOperatorReportSnapshotV2) error

	// DeleteSnapshot removes a report snapshot object from the database
	DeleteSnapshot(ctx context.Context, id string) error
}

// New returns an instance of DataStore.
func New(reportSnapshotStorage pgStore.Store) DataStore {
	return &datastoreImpl{
		store: reportSnapshotStorage,
	}
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) DataStore {
	return New(pgStore.New(pool))
}
