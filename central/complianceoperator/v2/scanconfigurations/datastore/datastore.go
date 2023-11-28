package datastore

import (
	"context"
	"testing"

	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore/search"
	statusStore "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/scanconfigstatus/store/postgres"
	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/postgres"
)

// DataStore is the entry point for storing/retrieving compliance operator metadata.
//
//go:generate mockgen-wrapper
type DataStore interface {
	// GetScanConfiguration retrieves the scan configuration specified by id
	GetScanConfiguration(ctx context.Context, id string) (*storage.ComplianceOperatorScanConfigurationV2, bool, error)

	// ScanConfigurationExists retrieves the existence of scan configuration specified by name
	ScanConfigurationExists(ctx context.Context, scanName string) (bool, error)

	// GetScanConfigurations retrieves the scan configurations specified by query
	GetScanConfigurations(ctx context.Context, query *v1.Query) ([]*storage.ComplianceOperatorScanConfigurationV2, error)

	// UpsertScanConfiguration adds or updates the scan configuration
	UpsertScanConfiguration(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2) error

	// DeleteScanConfiguration deletes the scan configuration specified by id
	DeleteScanConfiguration(ctx context.Context, id string) (string, []string, error)

	// UpdateClusterStatus updates the scan configuration with the cluster status
	UpdateClusterStatus(ctx context.Context, scanID string, clusterID string, clusterStatus string) error

	// GetScanConfigClusterStatus retrieves the scan configurations status per cluster specified by scan id
	GetScanConfigClusterStatus(ctx context.Context, scanID string) ([]*storage.ComplianceOperatorClusterScanConfigStatus, error)

	// Count scan config based on a query
	CountScanConfigurations(ctx context.Context, q *v1.Query) (int, error)
}

// New returns an instance of DataStore.
func New(scanConfigStore pgStore.Store, scanConfigStatusStore statusStore.Store, clusterDS clusterDatastore.DataStore, searcher search.Searcher) DataStore {
	ds := &datastoreImpl{
		storage:       scanConfigStore,
		statusStorage: scanConfigStatusStore,
		clusterDS:     clusterDS,
		keyedMutex:    concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize),
		searcher:      searcher,
	}
	return ds
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool postgres.DB, clusterDS clusterDatastore.DataStore, searcher search.Searcher) (DataStore, error) {
	store := pgStore.New(pool)
	statusStore := statusStore.New(pool)
	return New(store, statusStore, clusterDS, searcher), nil
}
