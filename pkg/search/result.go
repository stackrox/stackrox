package search

import (
	"github.com/stackrox/rox/generated/api/v1"
)

// GetProtoMatchesMap offloads the values of the input map into SeachResult_Matches types.
func GetProtoMatchesMap(m map[string][]string) map[string]*v1.SearchResult_Matches {
	matches := make(map[string]*v1.SearchResult_Matches)
	for k, v := range m {
		matches[k] = &v1.SearchResult_Matches{Values: v}
	}
	return matches
}
