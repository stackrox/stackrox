package evaluator

import (
	"github.com/stackrox/stackrox/pkg/booleanpolicy/evaluator/pathutil"
)

// A Match represents a single match.
// It contains the matched value, as well as the path
// within the object that was taken to reach the value.
type Match struct {
	Path *pathutil.Path
	// Values represents a list of human-friendly representations of the matched value.
	Values []string
}

// GetPath implements the pathutil.PathHolder interface.
func (m Match) GetPath() *pathutil.Path {
	return m.Path
}

// GetValues implements the pathutil.PathHolder interface.
func (m Match) GetValues() []string {
	return m.Values
}

// A fieldResult is the result of evaluating a query on an object.
type fieldResult struct {
	Matches map[string][]Match
}

// A Result is the result of evaluating a query on an object, with only linked matches.
type Result struct {
	Matches []map[string][]string
}

func mergeResults(results []*fieldResult) *fieldResult {
	if len(results) == 0 {
		return nil
	}

	merged := &fieldResult{Matches: make(map[string][]Match)}
	for _, r := range results {
		for field, matches := range r.Matches {
			merged.Matches[field] = append(merged.Matches[field], matches...)
		}
	}
	return merged
}
