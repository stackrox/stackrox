package types

import (
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

// Hash provides an integer value for the GetFlags flag set.
func (f GetFlags) Hash() int32 {
	return int32(f)
}

// ResultsWithStatus returns the last successful results, as well as the metadata for the recent (i.e., since the
// last successful results) failed results.
type ResultsWithStatus struct {
	LastSuccessfulResults *storage.ComplianceRunResults
	FailedRuns            []*storage.ComplianceRunMetadata
}

// ComplianceRunsMetadata returns the metadata for the last successful compliance run, as well as metadata for the
// recent (i.e., since the last successful results) failed results.
type ComplianceRunsMetadata struct {
	LastSuccessfulRunMetadata *storage.ComplianceRunMetadata
	FailedRunsMetadata        []*storage.ComplianceRunMetadata
}
