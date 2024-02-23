package datastore

import (
	"context"

	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/scansettingbindings/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore is the entry point for storing/retrieving compliance scan setting binding objects
//
//go:generate mockgen-wrapper
type DataStore interface {
	// GetScanSettingBinding retrieves the scan setting binding object from the database
	GetScanSettingBinding(ctx context.Context, id string) (*storage.ComplianceOperatorScanSettingBindingV2, bool, error)

	// UpsertScanSettingBinding adds the scan setting binding object to the database
	UpsertScanSettingBinding(ctx context.Context, result *storage.ComplianceOperatorScanSettingBindingV2) error

	// DeleteScanSettingBinding removes a scan setting binding object from the database
	DeleteScanSettingBinding(ctx context.Context, id string) error

	// GetScanSettingBindingsByCluster retrieves scan setting bindings by cluster
	GetScanSettingBindingsByCluster(ctx context.Context, clusterID string) ([]*storage.ComplianceOperatorScanSettingBindingV2, error)

	// GetScanSettingBindings retrieves scan setting bindings matching the query
	GetScanSettingBindings(ctx context.Context, query *v1.Query) ([]*storage.ComplianceOperatorScanSettingBindingV2, error)
}

// New returns an instance of DataStore.
func New(scanSettingBindingStorage pgStore.Store) DataStore {
	return &datastoreImpl{
		store: scanSettingBindingStorage,
	}
}
