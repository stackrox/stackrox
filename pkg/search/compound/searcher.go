package compound

import (
	"context"

	"github.com/gogo/protobuf/proto"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/dackbox/keys/transformation"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
)

// SearcherSpec specifies a searcher for the compound searcher to use.
type SearcherSpec struct {
	IsDefault bool
	Searcher  search.Searcher
	Options   search.OptionsMap

	// If you want a transformation applied to the resultss prior to aggregation.
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
func NewSearcher(specs []SearcherSpec, names ...string) search.Searcher {
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
	var local *v1.Query
	if q != nil {
		local = proto.Clone(q).(*v1.Query)
	}

	// Construct a tree that matches subqueries with specifications.
	req, err := build(local, cs.specs)
	if err != nil {
		return nil, err
	}

	// Optimize the tree by combining subtrees that reference the same searcher specification.
	condensed, err := condense(req)
	if err != nil {
		return nil, err
	}

	// Add the sorting as necessary to the condensed tree.
	sorted, err := addSorting(condensed, local.GetPagination(), cs.specs)
	if err != nil {
		return nil, err
	}

	// Execute the tree.
	return execute(ctx, sorted)
}
