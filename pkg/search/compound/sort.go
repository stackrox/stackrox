package compound

import (
	"errors"

	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/search"
)

func addSorting(rs *searchRequestSpec, p *v1.QueryPagination, specs []SearcherSpec) (*searchRequestSpec, error) {
	// No pagination no problems.
	if p == nil {
		return rs, nil
	}

	// If the top level is a base query on the same searcher, we can stuff pagination there.
	if ret, sorted := trySortBase(rs, p); sorted {
		return ret, nil
	}

	// If the top level is a conjunction query, and one of those conjuncts is a base query on the same searcher we are paginating
	// with, we can stuff it there.
	if ret, sorted := trySortAnd(rs, p); sorted {
		return ret, nil
	}

	// Otherwise, we need to add a layer.
	if ret, sorted := trySortComplex(rs, p, specs); sorted {
		return ret, nil
	}
	return nil, errors.New("pagination does not match any searcher spec")
}

// We can just add sorting info if the top level is a single query on a single searcher.
func trySortBase(rs *searchRequestSpec, p *v1.QueryPagination) (*searchRequestSpec, bool) {
	if rs.base != nil && paginationMatchesOptions(p, rs.base.Spec.Options) {
		rs.base.Query.Pagination = p
		return rs, true
	}
	return nil, false
}

// If the top level is a Conjunction/And, then we can add the sorting there.
func trySortAnd(rs *searchRequestSpec, p *v1.QueryPagination) (*searchRequestSpec, bool) {
	if len(rs.and) > 0 {
		for _, sr := range rs.and {
			if sr.base != nil && paginationMatchesOptions(p, sr.base.Spec.Options) {
				sr.base.Query.Pagination = p
				return rs, true
			}
		}
	}
	return nil, false
}

// If the top level isn't a base nor a conjunction with a base that operated on the searcher we want to sort with,
// then we need to do the sorting as a separate query.
func trySortComplex(rs *searchRequestSpec, p *v1.QueryPagination, specs []SearcherSpec) (*searchRequestSpec, bool) {
	// Add a layer with an and, and sort.
	for i := range specs {
		spec := specs[i]
		if paginationMatchesOptions(p, spec.Options) {
			q := search.EmptyQuery()
			q.Pagination = p
			return &searchRequestSpec{
				leftJoinWithRightOrder: &joinRequestSpec{
					left: rs,
					right: &searchRequestSpec{
						base: &baseRequestSpec{
							Spec:  &spec,
							Query: q,
						},
					},
				},
			}, true
		}
	}
	return nil, false
}

// For simplicity's sake, we only allow sorting from a single child searcher. If we are sorting on fields across
// multiple, it would be complicated to implement, so update this if you need that.
func paginationMatchesOptions(p *v1.QueryPagination, merp search.OptionsMap) bool {
	for _, so := range p.GetSortOptions() {
		if _, matches := merp.Get(so.GetField()); !matches {
			return false
		}
	}
	return true
}
