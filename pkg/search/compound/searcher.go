package compound

import (
	"context"

	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/dackbox/keys/transformation"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/paginated"
)

// SearcherSpec specifies a searcher for the compound searcher to use.
type SearcherSpec struct {
	IsDefault bool
	Searcher  search.Searcher
	Options   search.OptionsMap

	// If you want a transformation applied to the results prior to aggregation.
	Transformation transformation.OneToMany

	// Provides the ability to do linked fields queries. You should populate this field with a transformation that
	// converts an id from the following searcher to an id of this searchers type.
	// For instance, if you wanted to do a linked field search on component and cve fields:
	// NewSearcher(
	//    &SearcherSpec{ Searcher: cveSearcher }
	//    // Can do a linked field search with component and CVE fields.
	//    &SearcherSpec{ Searcher: componentSearcher LinkToPrev: <Mapping from CVE -> Components>}
	//    // No LinkToPrev, so you can't do a linked search with Image fields and component fields
	//     &SearcherSpec{ Searcher: imageSearcher }
	// )
	LinkToPrev transformation.OneToMany
}

// NewSearcher returns a searcher that applies search terms to the first input index that supports the term.
// If no index supports the term, then the search will return an error.
func NewSearcher(specs []SearcherSpec) search.Searcher {
	return paginated.Paginated(&compoundSearcherImpl{
		specs: specs,
	})
}

type compoundSearcherImpl struct {
	specs []SearcherSpec
}

// Search constructs and executes the necessary queries on the searchers that the compound searcher is configured to
// use.
func (cs *compoundSearcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return cs.searchInternal(ctx, q, false)
}

// Count uses Search function to get the search results and then count them
func (cs *compoundSearcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	results, err := cs.searchInternal(ctx, q, true)
	return len(results), err
}

func (cs *compoundSearcherImpl) searchInternal(ctx context.Context, q *v1.Query, skipSort bool) ([]search.Result, error) {
	var local *v1.Query
	if q != nil {
		local = q.Clone()
	}

	// Construct a tree that matches subqueries with specifications.
	req, err := build(local, cs.specs)
	if err != nil {
		return nil, err
	}

	// both req and err will be nil if there is an unhandled option trying to be searched
	// e.g. Policy on deployments or images
	if req == nil {
		return nil, nil
	}

	// Optimize the tree by combining subtrees that reference the same searcher specification.
	condensed, err := condense(req)
	if err != nil {
		return nil, err
	}

	if skipSort {
		if local.GetPagination().GetSortOptions() != nil {
			local.Pagination.SortOptions = nil
		}
	}

	// Add the sorting as necessary to the condensed tree.
	sorted, err := addSorting(condensed, local.GetPagination(), cs.specs)
	if err != nil {
		return nil, err
	}

	// Execute the tree.
	return execute(ctx, sorted)
}
