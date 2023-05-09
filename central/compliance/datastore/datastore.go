package datastore

import (
	"context"

	"github.com/stackrox/rox/central/compliance"
	"github.com/stackrox/rox/central/compliance/datastore/internal/store"
	"github.com/stackrox/rox/central/compliance/datastore/types"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore is the interface for accessing stored compliance data
//
//go:generate mockgen-wrapper
type DataStore interface {
	GetSpecificRunResults(ctx context.Context, clusterID, standardID, runID string, flags types.GetFlags) (types.ResultsWithStatus, error)
	GetLatestRunResults(ctx context.Context, clusterID, standardID string, flags types.GetFlags) (types.ResultsWithStatus, error)
	GetLatestRunResultsBatch(ctx context.Context, clusterIDs, standardIDs []string, flags types.GetFlags) (map[compliance.ClusterStandardPair]types.ResultsWithStatus, error)
	IsComplianceRunSuccessfulOnCluster(ctx context.Context, clusterID string, standardIDs []string) (bool, error)

	StoreRunResults(ctx context.Context, results *storage.ComplianceRunResults) error
	StoreFailure(ctx context.Context, metadata *storage.ComplianceRunMetadata) error
	StoreComplianceDomain(ctx context.Context, domain *storage.ComplianceDomain) error

	UpdateConfig(ctx context.Context, id string, hide bool) error
	GetConfig(ctx context.Context, id string) (*storage.ComplianceConfig, bool, error)

	PerformStoredAggregation(ctx context.Context, args *StoredAggregationArgs) ([]*storage.ComplianceAggregation_Result, []*storage.ComplianceAggregation_Source, map[*storage.ComplianceAggregation_Result]*storage.ComplianceDomain, error)
	ClearAggregationResults(ctx context.Context) error
}

// StoredAggregationArgs encapsulates the arguments to the PerformStoredAggregation method
type StoredAggregationArgs struct {
	QueryString     string
	GroupBy         []storage.ComplianceAggregation_Scope
	Unit            storage.ComplianceAggregation_Scope
	AggregationFunc AggregationFunc
}

// AggregationFunc is a function that returns the results of a compliance aggregation
type AggregationFunc func() ([]*storage.ComplianceAggregation_Result, []*storage.ComplianceAggregation_Source, map[*storage.ComplianceAggregation_Result]*storage.ComplianceDomain, error)

// NewDataStore returns a new instance of a DataStore.
func NewDataStore(storage store.Store, filter SacFilter) DataStore {
	return &datastoreImpl{
		storage: storage,
		filter:  filter,
	}
}
