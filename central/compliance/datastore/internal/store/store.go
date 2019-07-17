package store

import (
	"github.com/stackrox/rox/central/compliance"
	"github.com/stackrox/rox/central/compliance/datastore/types"
	"github.com/stackrox/rox/generated/storage"
)

// Store is the interface for accessing stored compliance data
//go:generate mockgen-wrapper Store
type Store interface {
	GetLatestRunResults(clusterID, standardID string, flags types.GetFlags) (types.ResultsWithStatus, error)
	GetLatestRunResultsBatch(clusterIDs, standardIDs []string, flags types.GetFlags) (map[compliance.ClusterStandardPair]types.ResultsWithStatus, error)
	GetLatestRunResultsFiltered(clusterIDFilter, standardIDFilter func(string) bool, flags types.GetFlags) (map[compliance.ClusterStandardPair]types.ResultsWithStatus, error)
	GetLatestRunMetadataBatch(clusterID string, standardIDs []string) (map[compliance.ClusterStandardPair]types.ComplianceRunsMetadata, error)
	StoreRunResults(results *storage.ComplianceRunResults) error
	StoreFailure(metadata *storage.ComplianceRunMetadata) error
}
