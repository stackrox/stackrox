package datastore

import (
	"context"
	"testing"

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

	// GetScanConfigurationByName retrieves the scan configuration specified by name
	GetScanConfigurationByName(ctx context.Context, scanName string) (*storage.ComplianceOperatorScanConfigurationV2, error)

	// ScanConfigurationProfileExists takes all the profiles being referenced by the scan configuration and checks if any cluster is using it in any existing scan configurations.
	ScanConfigurationProfileExists(ctx context.Context, id string, profiles []string, clusters []string) (bool, error)

	// GetScanConfigurations retrieves the scan configurations specified by query
	GetScanConfigurations(ctx context.Context, query *v1.Query) ([]*storage.ComplianceOperatorScanConfigurationV2, error)

	// UpsertScanConfiguration adds or updates the scan configuration
	UpsertScanConfiguration(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2) error

	// DeleteScanConfiguration deletes the scan configuration specified by id
	DeleteScanConfiguration(ctx context.Context, id string) (string, error)

	// UpdateClusterStatus updates the scan configuration with the cluster status
	UpdateClusterStatus(ctx context.Context, scanConfigID string, clusterID string, clusterStatus string, clusterName string) error

	// GetScanConfigClusterStatus retrieves the scan configurations status per cluster specified by scan id
	GetScanConfigClusterStatus(ctx context.Context, scanConfigID string) ([]*storage.ComplianceOperatorClusterScanConfigStatus, error)

	// CountScanConfigurations scan config based on a query
	CountScanConfigurations(ctx context.Context, q *v1.Query) (int, error)

	// Remove deleted cluster from scan config
	RemoveClusterFromScanConfig(ctx context.Context, clusterID string) error
}

// New returns an instance of DataStore.
func New(scanConfigStore pgStore.Store, scanConfigStatusStore statusStore.Store) DataStore {
	ds := &datastoreImpl{
		storage:       scanConfigStore,
		statusStorage: scanConfigStatusStore,
		keyedMutex:    concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize),
	}
	return ds
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool postgres.DB) (DataStore, error) {
	store := pgStore.New(pool)
	statusStorage := statusStore.New(pool)
	return New(store, statusStorage), nil
}
