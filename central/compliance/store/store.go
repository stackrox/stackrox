package store

import (
	"github.com/stackrox/rox/central/compliance"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// GetFlags controls the behavior of the Get... methods of a Store.
type GetFlags int32

const (
	// WithMessageStrings will cause compliance results to be loaded with message strings.
	WithMessageStrings GetFlags = 1 << iota
	// RequireMessageStrings implies WithMessageStrings, and additionally fails with an error if any message strings
	// could not be loaded.
	RequireMessageStrings
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

	GetLatestRunResults(clusterID, standardID string, flags GetFlags) (ResultsWithStatus, error)
	GetLatestRunResultsBatch(clusterIDs, standardIDs []string, flags GetFlags) (map[compliance.ClusterStandardPair]ResultsWithStatus, error)
	GetLatestRunResultsFiltered(clusterIDFilter, standardIDFilter func(string) bool, flags GetFlags) (map[compliance.ClusterStandardPair]ResultsWithStatus, error)

	StoreRunResults(results *storage.ComplianceRunResults) error
	StoreFailure(metadata *storage.ComplianceRunMetadata) error
}

//go:generate mockgen-wrapper Store
