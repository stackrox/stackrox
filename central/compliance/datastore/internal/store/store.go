package store

import (
	"context"

	"github.com/stackrox/rox/central/compliance"
	"github.com/stackrox/rox/central/compliance/datastore/types"
	"github.com/stackrox/rox/generated/storage"
)

// Store is the interface for accessing stored compliance data
//
//go:generate mockgen-wrapper
type Store interface {
	GetSpecificRunResults(ctx context.Context, clusterID, standardID, runID string, flags types.GetFlags) (types.ResultsWithStatus, error)
	GetLatestRunResults(ctx context.Context, clusterID, standardID string, flags types.GetFlags) (types.ResultsWithStatus, error)
	GetLatestRunResultsBatch(ctx context.Context, clusterIDs, standardIDs []string, flags types.GetFlags) (map[compliance.ClusterStandardPair]types.ResultsWithStatus, error)
	GetLatestRunMetadataBatch(ctx context.Context, clusterID string, standardIDs []string) (map[compliance.ClusterStandardPair]types.ComplianceRunsMetadata, error)
	StoreRunResults(ctx context.Context, results *storage.ComplianceRunResults) error
	StoreFailure(ctx context.Context, metadata *storage.ComplianceRunMetadata) error
	StoreComplianceDomain(ctx context.Context, domain *storage.ComplianceDomain) error
	StoreAggregationResult(ctx context.Context, queryString string, groupBy []storage.ComplianceAggregation_Scope, unit storage.ComplianceAggregation_Scope, results []*storage.ComplianceAggregation_Result, sources []*storage.ComplianceAggregation_Source, domains map[*storage.ComplianceAggregation_Result]*storage.ComplianceDomain) error
	UpdateConfig(ctx context.Context, config *storage.ComplianceConfig) error
	GetConfig(ctx context.Context, id string) (*storage.ComplianceConfig, bool, error)
	GetAggregationResult(ctx context.Context, queryString string, groupBy []storage.ComplianceAggregation_Scope, unit storage.ComplianceAggregation_Scope) ([]*storage.ComplianceAggregation_Result, []*storage.ComplianceAggregation_Source, map[*storage.ComplianceAggregation_Result]*storage.ComplianceDomain, error)
	ClearAggregationResults(ctx context.Context) error
}
