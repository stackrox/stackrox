package compound

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
)

// SearcherSpec specifies a searcher for the compound searcher to use.
type SearcherSpec struct {
	IsDefault   bool
	DropHandled bool
	Searcher    search.Searcher
	Options     search.OptionsMap
}

// NewSearcher returns a searcher that applies search terms to the first input index that supports the term.
// If no index supports the term, then the search will return an error.
func NewSearcher(specs ...SearcherSpec) search.Searcher {
	optMaps := make([]search.OptionsMap, 0, len(specs))
	for _, spec := range specs {
		optMaps = append(optMaps, spec.Options)
	}
	return paginated.Paginated(&compoundSearcherImpl{
		specs:    specs,
		combined: search.CombineOptionsMaps(optMaps...),
	})
}

type compoundSearcherImpl struct {
	specs    []SearcherSpec
	combined search.OptionsMap
}

// Search constructs and executes the necessary queries on the searchers that the compound searcher is configured to
// use.
func (cs *compoundSearcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	// Filter unsupported fields.
	p := q.GetPagination()
	q, _ = search.FilterQueryWithMap(q, cs.combined)
	if q == nil {
		q = search.EmptyQuery()
	}
	q.Pagination = p

	// Construct a tree that matches subqueries with specifications.
	req, err := build(q, cs.specs)
	if err != nil {
		return nil, err
	}

	// Optimize the tree by combining subtrees that reference the same searcher specification.
	condensed, err := condense(req)
	if err != nil {
		return nil, err
	}

	// Add the sorting as necessary to the condensed tree.
	sorted, err := addSorting(condensed, q.GetPagination(), cs.specs)
	if err != nil {
		return nil, err
	}

	// Execute the tree.
	return execute(ctx, sorted)
}
