package store

import (
	"github.com/stackrox/stackrox/central/compliance"
	"github.com/stackrox/stackrox/central/compliance/datastore/types"
	"github.com/stackrox/stackrox/generated/storage"
)

// Store is the interface for accessing stored compliance data
//go:generate mockgen-wrapper
type Store interface {
	GetSpecificRunResults(clusterID, standardID, runID string, flags types.GetFlags) (types.ResultsWithStatus, error)
	GetLatestRunResults(clusterID, standardID string, flags types.GetFlags) (types.ResultsWithStatus, error)
	GetLatestRunResultsBatch(clusterIDs, standardIDs []string, flags types.GetFlags) (map[compliance.ClusterStandardPair]types.ResultsWithStatus, error)
	GetLatestRunMetadataBatch(clusterID string, standardIDs []string) (map[compliance.ClusterStandardPair]types.ComplianceRunsMetadata, error)
	StoreRunResults(results *storage.ComplianceRunResults) error
	StoreFailure(metadata *storage.ComplianceRunMetadata) error
	StoreComplianceDomain(domain *storage.ComplianceDomain) error
	StoreAggregationResult(queryString string, groupBy []storage.ComplianceAggregation_Scope, unit storage.ComplianceAggregation_Scope, results []*storage.ComplianceAggregation_Result, sources []*storage.ComplianceAggregation_Source, domains map[*storage.ComplianceAggregation_Result]*storage.ComplianceDomain) error
	GetAggregationResult(queryString string, groupBy []storage.ComplianceAggregation_Scope, unit storage.ComplianceAggregation_Scope) ([]*storage.ComplianceAggregation_Result, []*storage.ComplianceAggregation_Source, map[*storage.ComplianceAggregation_Result]*storage.ComplianceDomain, error)
	ClearAggregationResults() error
}
