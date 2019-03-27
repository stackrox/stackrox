package store

import (
	"github.com/stackrox/rox/central/compliance"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// ValidResultsAndSources decomposes the given map into the valid, most recent results, and a list of sources that store the
// metadata of all referenced cluster/standard pair and their successful/failed runs.
func ValidResultsAndSources(allResults map[compliance.ClusterStandardPair]ResultsWithStatus) ([]*storage.ComplianceRunResults, []*v1.ComplianceAggregation_Source) {
	var validResults []*storage.ComplianceRunResults
	var sources []*v1.ComplianceAggregation_Source

	for key, res := range allResults {
		if res.LastSuccessfulResults != nil {
			validResults = append(validResults, res.LastSuccessfulResults)
		}
		source := &v1.ComplianceAggregation_Source{
			ClusterId:     key.ClusterID,
			StandardId:    key.StandardID,
			SuccessfulRun: res.LastSuccessfulResults.GetRunMetadata(), // nil-safe
			FailedRuns:    res.FailedRuns,
		}
		sources = append(sources, source)
	}
	return validResults, sources
}
