package search

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/set"
)

// Result is a wrapper around the search results
type Result struct {
	ID      string
	Matches map[string][]string
	Score   float64

	// NEW: Optional fields populated when selects are provided in query
	// These fields enable single-pass query construction of SearchResult protos
	Name        string            // Display name (e.g., image name, deployment name)
	Location    string            // Human-readable location (e.g., cluster/namespace/name)
	Category    v1.SearchCategory // Search result category
	FieldValues map[string]string // Selected field values as strings, keyed by field name
}

// CountByWrapper wraps around the result of a CountBy query.
// It stores the values of the count by field tuples in ByFields and the related count in Count.
type CountByWrapper struct {
	ByFields Result
	Count    int
}

// NewResult returns a new search result
func NewResult() *Result {
	return &Result{
		Matches: make(map[string][]string),
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

// SearchResultConverter defines how to build a SearchResult proto from a Result.
// This interface allows datastores to customize name and location construction.
type SearchResultConverter interface {
	// BuildName extracts and formats the name field from a Result
	BuildName(result *Result) string

	// BuildLocation builds the location string from a Result
	BuildLocation(result *Result) string

	// GetCategory returns the SearchCategory for this result type
	GetCategory() v1.SearchCategory
}

// ResultsToSearchResultProtos converts search Results to SearchResult protos using a converter.
// This eliminates the need for 2-pass queries by building the proto directly from selected fields.
func ResultsToSearchResultProtos(results []Result, converter SearchResultConverter) []*v1.SearchResult {
	protoResults := make([]*v1.SearchResult, 0, len(results))
	for i := range results {
		result := &results[i]
		protoResults = append(protoResults, &v1.SearchResult{
			Id:             result.ID,
			Name:           converter.BuildName(result),
			Location:       converter.BuildLocation(result),
			Category:       converter.GetCategory(),
			FieldToMatches: GetProtoMatchesMap(result.Matches),
			Score:          result.Score,
		})
	}
	return protoResults
}

// DefaultSearchResultConverter provides a generic implementation for simple cases
// where Name and Location are directly available in the Result.
type DefaultSearchResultConverter struct {
	Category v1.SearchCategory
}

// BuildName returns the Name field directly
func (c *DefaultSearchResultConverter) BuildName(result *Result) string {
	return result.Name
}

// BuildLocation returns the Location field directly
func (c *DefaultSearchResultConverter) BuildLocation(result *Result) string {
	return result.Location
}

// GetCategory returns the configured category
func (c *DefaultSearchResultConverter) GetCategory() v1.SearchCategory {
	return c.Category
}
