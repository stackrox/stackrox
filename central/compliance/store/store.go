package store

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Store is the interface for accessing stored compliance data
type Store interface {
	QueryControlResults(query *v1.Query) ([]*storage.ComplianceControlResult, error)

	GetLatestRunResults(clusterID, standardID string) (*storage.ComplianceRunResults, error)
	GetLatestRunResultsBatch(clusterIDs, standardIDs []string) ([]*storage.ComplianceRunResults, error)
	GetLatestRunResultsFiltered(clusterIDFilter, standardIDFilter func(string) bool) ([]*storage.ComplianceRunResults, error)

	StoreRunResults(results *storage.ComplianceRunResults) error
}

//go:generate mockgen-wrapper Store
