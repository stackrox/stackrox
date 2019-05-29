package blevesearch

import (
	"context"

	"github.com/blevesearch/bleve"
	v1 "github.com/stackrox/rox/generated/api/v1"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// Searcher is a search.Searcher implementation with nothing but bleve underneath.
type Searcher struct {
	Index      bleve.Index
	Category   v1.SearchCategory
	OptionsMap searchPkg.OptionsMap
}

// Search builds a query and runs it against the index.
func (s *Searcher) Search(_ context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return RunSearchRequest(s.Category, q, s.Index, s.OptionsMap)
}
