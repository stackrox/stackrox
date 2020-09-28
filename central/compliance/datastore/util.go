package datastore

import (
	"github.com/stackrox/rox/central/compliance"
	"github.com/stackrox/rox/central/compliance/datastore/types"
	"github.com/stackrox/rox/generated/storage"
)

// ValidResultsAndSources decomposes the given map into the valid, most recent results, and a list of sources that store the
// metadata of all referenced cluster/standard pair and their successful/failed runs.
func ValidResultsAndSources(allResults map[compliance.ClusterStandardPair]types.ResultsWithStatus) ([]*storage.ComplianceRunResults, []*storage.ComplianceAggregation_Source) {
	var validResults []*storage.ComplianceRunResults
	var sources []*storage.ComplianceAggregation_Source

	for key, res := range allResults {
		if res.LastSuccessfulResults != nil {
			validResults = append(validResults, res.LastSuccessfulResults)
		}
		source := &storage.ComplianceAggregation_Source{
			ClusterId:     key.ClusterID,
			StandardId:    key.StandardID,
			SuccessfulRun: res.LastSuccessfulResults.GetRunMetadata(), // nil-safe
			FailedRuns:    res.FailedRuns,
		}
		sources = append(sources, source)
	}
	return validResults, sources
}
