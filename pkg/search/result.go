package search

import (
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/set"
)

// Result is a wrapper around the search results
type Result struct {
	ID      string
	Matches map[string][]string
	Score   float64
	Fields  map[string]interface{}
}

// NewResult returns a new search result
func NewResult() *Result {
	return &Result{
		Matches: make(map[string][]string),
		Fields:  make(map[string]interface{}),
	}
}

// GetProtoMatchesMap offloads the values of the input map into SearchResult_Matches types.
func GetProtoMatchesMap(m map[string][]string) map[string]*v1.SearchResult_Matches {
	matches := make(map[string]*v1.SearchResult_Matches)
	for k, v := range m {
		matches[k] = &v1.SearchResult_Matches{Values: v}
	}
	return matches
}

// ResultsToIDs takes a results slice and gets a slice of just the IDs
func ResultsToIDs(results []Result) []string {
	ids := make([]string, 0, len(results))
	for _, r := range results {
		ids = append(ids, r.ID)
	}
	return ids
}

// ResultsToIDSet takes a results slice and gets a set of IDs
func ResultsToIDSet(results []Result) set.StringSet {
	ids := set.NewStringSet()
	for _, r := range results {
		ids.Add(r.ID)
	}
	return ids
}

// RemoveMissingResults removes those indices in the result set that are specified in the missingIndices
// slice. The missingIndices slice MUST be sorted.
func RemoveMissingResults(results []Result, missingIndices []int) []Result {
	numResultsBefore := len(results)

	var outIdx int
	for i := 0; i < len(missingIndices); i++ {
		missingIdx := missingIndices[i]
		if i == 0 {
			outIdx = missingIdx
		}
		rangeBegin := missingIdx + 1
		rangeEnd := numResultsBefore
		if i+1 < len(missingIndices) {
			rangeEnd = missingIndices[i+1]
		}
		chunkSize := rangeEnd - rangeBegin
		if chunkSize == 0 {
			continue
		}
		copy(results[outIdx:outIdx+chunkSize], results[rangeBegin:rangeEnd])
		outIdx += chunkSize
	}
	return results[:numResultsBefore-len(missingIndices)]
}
