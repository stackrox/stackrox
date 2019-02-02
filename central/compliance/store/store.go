package store

import (
	"github.com/stackrox/rox/central/compliance"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// ResultsWithStatus returns the last successful results, as well as the metadata for the recent (i.e., since the
// last successful results) failed results.
type ResultsWithStatus struct {
	LastSuccessfulResults *storage.ComplianceRunResults
	FailedRuns            []*storage.ComplianceRunMetadata
}

// Store is the interface for accessing stored compliance data
type Store interface {
	QueryControlResults(query *v1.Query) ([]*storage.ComplianceControlResult, error)

	GetLatestRunResults(clusterID, standardID string) (ResultsWithStatus, error)
	GetLatestRunResultsBatch(clusterIDs, standardIDs []string) (map[compliance.ClusterStandardPair]ResultsWithStatus, error)
	GetLatestRunResultsFiltered(clusterIDFilter, standardIDFilter func(string) bool) (map[compliance.ClusterStandardPair]ResultsWithStatus, error)

	StoreRunResults(results *storage.ComplianceRunResults) error
	StoreFailure(metadata *storage.ComplianceRunMetadata) error
}

//go:generate mockgen-wrapper Store
