package search

import v1 "github.com/stackrox/rox/generated/api/v1"

// Searcher allows you to search objects.
type Searcher interface {
	Search(q *v1.Query) ([]Result, error)
}
