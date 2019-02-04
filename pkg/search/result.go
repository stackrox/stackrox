package search

import (
	"github.com/stackrox/rox/generated/api/v1"
)

// Result is a wrapper around the search results
type Result struct {
	ID      string
	Matches map[string][]string
	Score   float64
	Fields  map[string]interface{}
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
