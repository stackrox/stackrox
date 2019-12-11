package datastore

import (
	"context"

	"github.com/stackrox/rox/central/compliance"
	"github.com/stackrox/rox/central/compliance/datastore/internal/store"
	"github.com/stackrox/rox/central/compliance/datastore/types"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore is the interface for accessing stored compliance data
//go:generate mockgen-wrapper DataStore
type DataStore interface {
	QueryControlResults(ctx context.Context, query *v1.Query) ([]*storage.ComplianceControlResult, error)

	GetSpecificRunResults(ctx context.Context, clusterID, standardID, runID string, flags types.GetFlags) (types.ResultsWithStatus, error)
	GetLatestRunResults(ctx context.Context, clusterID, standardID string, flags types.GetFlags) (types.ResultsWithStatus, error)
	GetLatestRunResultsBatch(ctx context.Context, clusterIDs, standardIDs []string, flags types.GetFlags) (map[compliance.ClusterStandardPair]types.ResultsWithStatus, error)
	GetLatestRunResultsFiltered(ctx context.Context, clusterIDFilter, standardIDFilter func(string) bool, flags types.GetFlags) (map[compliance.ClusterStandardPair]types.ResultsWithStatus, error)
	GetLatestRunMetadataBatch(ctx context.Context, clusterID string, standardIDs []string) (map[compliance.ClusterStandardPair]types.ComplianceRunsMetadata, error)
	IsComplianceRunSuccessfulOnCluster(ctx context.Context, clusterID string, standardIDs []string) (bool, error)

	StoreRunResults(ctx context.Context, results *storage.ComplianceRunResults) error
	StoreFailure(ctx context.Context, metadata *storage.ComplianceRunMetadata) error
}

// NewDataStore returns a new instance of a DataStore.
func NewDataStore(storage store.Store, filter SacFilter) DataStore {
	return &datastoreImpl{
		storage: storage,
		filter:  filter,
	}
}
