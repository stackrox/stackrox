package search

import (
	"fmt"
	"math"
	"strconv"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/set"
)

// GetValueFromField returns a string value from a search result field
func GetValueFromField(val interface{}) string {
	switch val := val.(type) {
	case string:
		return val
	case float64:
		i, f := math.Modf(val)
		// If it's an int, return just the int portion.
		if math.Abs(f) < 1e-3 {
			return fmt.Sprintf("%d", int(i))
		}
		return fmt.Sprintf("%.2f", val)
	case bool:
		return strconv.FormatBool(val)
	default:
		log.Errorf("Unknown type field from index: %T", val)
	}
	return ""
}

// GetValuesFromFields returns the values from the given field as a string slice.
func GetValuesFromFields(field string, m map[string]interface{}) []string {
	val, ok := m[field]
	if !ok {
		return nil
	}

	if asSlice, ok := val.([]interface{}); ok {
		values := make([]string, 0, len(asSlice))
		for _, v := range asSlice {
			strVal := GetValueFromField(v)
			if strVal != "" {
				values = append(values, strVal)
			}
		}
		return values
	}

	strVal := GetValueFromField(val)
	if strVal == "" {
		return nil
	}
	return []string{strVal}
}

// Result is a wrapper around the search results
type Result struct {
	ID      string
	Matches map[string][]string
	Score   float64
	Fields  map[string]interface{}
}

// GetValuesFromFieldPath returns a slice of strings from the result fields
// for the passed field path
func (r Result) GetValuesFromFieldPath(fieldPath string) []string {
	return GetValuesFromFields(fieldPath, r.Fields)
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
