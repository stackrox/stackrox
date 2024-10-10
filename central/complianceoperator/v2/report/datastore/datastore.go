package datastore

import (
	"context"

	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/report/store/postgres"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore is the entry point for storing/retrieving compliance operator report snapshot objects.
//
//go:generate mockgen-wrapper
type DataStore interface {
	// GetSnapshot retrieves the report snapshot object from the database
	GetSnapshot(ctx context.Context, id string) (*storage.ComplianceOperatorReportSnapshotV2, bool, error)

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
