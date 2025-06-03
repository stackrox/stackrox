package datastore

import (
	"context"
	"testing"

	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/scans/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
)

// DataStore is the entry point for storing/retrieving compliance operator scan objects.
//
//go:generate mockgen-wrapper
type DataStore interface {
	// GetScan retrieves the scan object from the database
	GetScan(ctx context.Context, id string) (*storage.ComplianceOperatorScanV2, bool, error)

	// UpsertScan adds the scan object to the database
	UpsertScan(ctx context.Context, result *storage.ComplianceOperatorScanV2) error

	// DeleteScan removes a scan object from the database
	DeleteScan(ctx context.Context, id string) error

	// GetScansByCluster retrieves scan objects by cluster
	GetScansByCluster(ctx context.Context, clusterID string) ([]*storage.ComplianceOperatorScanV2, error)

	// DeleteScanByCluster deletes scans by cluster
	DeleteScanByCluster(ctx context.Context, clusterID string) error

	// SearchScans returns the scans for the given query
	SearchScans(ctx context.Context, query *v1.Query) ([]*storage.ComplianceOperatorScanV2, error)
}

// New returns an instance of DataStore.
func New(complianceScanStorage pgStore.Store) DataStore {
	return &datastoreImpl{
		store: complianceScanStorage,
	}
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) DataStore {
	store := pgStore.New(pool)
	return New(store)
}
