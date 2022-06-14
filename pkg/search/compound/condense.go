package compound

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// condense recursively condenses the execution tree.
// What this means is that any sub-tree that operate only on a single Searcher will be condensed into a single Query
// that operates on that searcher.
func condense(tree *searchRequestSpec) (*searchRequestSpec, error) {
	if len(tree.and) > 0 {
		return condenseAnd(tree)
	} else if len(tree.or) > 0 {
		return condenseOr(tree)
	} else if tree.boolean != nil {
		return condenseBoolean(tree)
	}
	return tree, nil
}

func condenseOr(req *searchRequestSpec) (*searchRequestSpec, error) {
	condensed, err := condenseList(req.or, search.DisjunctionQuery)
	if err != nil {
		return nil, err
	}
	if len(condensed) == 1 {
		return condensed[0], nil
	}
	return &searchRequestSpec{
		or: condensed,
	}, nil
}

func condenseAnd(req *searchRequestSpec) (*searchRequestSpec, error) {
	condensed, err := condenseList(req.and, search.ConjunctionQuery)
	if err != nil {
		return nil, err
	}
	if len(condensed) == 1 {
		return condensed[0], nil
	}
	return &searchRequestSpec{
		and: condensed,
	}, nil
}

func condenseBoolean(req *searchRequestSpec) (*searchRequestSpec, error) {
	must := req.boolean.must
	if len(must.and) > 0 {
		var err error
		must, err = condenseAnd(must)
		if err != nil || must == nil {
			return nil, err
		}
	}

	mustNot := req.boolean.mustNot
	if len(mustNot.or) > 0 {
		var err error
		mustNot, err = condenseOr(mustNot)
		if err != nil || mustNot == nil {
			return nil, err
		}
	}

	if mustNot.base == nil || must.base == nil || mustNot.base.Spec.Searcher != must.base.Spec.Searcher {
		return &searchRequestSpec{
			boolean: &booleanRequestSpec{
				must:    must,
				mustNot: mustNot,
			},
		}, nil
	}

	return &searchRequestSpec{
		base: &baseRequestSpec{
			Spec:  mustNot.base.Spec,
			Query: search.NewBooleanQuery(must.base.Query.GetConjunction(), mustNot.base.Query.GetDisjunction()),
		},
	}, nil
}

func condenseList(children []*searchRequestSpec, combineQueries func(q ...*v1.Query) *v1.Query) ([]*searchRequestSpec, error) {
	ret := make([]*searchRequestSpec, 0, len(children))
	specQueryIndex := make(map[*SearcherSpec]int)
	queriesPerSpec := make([][]*v1.Query, 0, len(children))
	for _, child := range children {
		condensed, err := condense(child)
		if err != nil {
			return nil, err
		}
		if condensed.base == nil {
			ret = append(ret, condensed)
			continue
		}

		index, hasIndex := specQueryIndex[condensed.base.Spec]
		if !hasIndex {
			index = len(queriesPerSpec)
			specQueryIndex[condensed.base.Spec] = index
			queriesPerSpec = append(queriesPerSpec, nil)
		}
		queriesPerSpec[index] = append(queriesPerSpec[index], condensed.base.Query)
	}
	if len(specQueryIndex) > 0 {
		ret = append(ret, condenseMap(specQueryIndex, queriesPerSpec, combineQueries)...)
	}
	return ret, nil
}

func condenseMap(specQueryIndex map[*SearcherSpec]int, queriesPerSpec [][]*v1.Query, combineQueries func(q ...*v1.Query) *v1.Query) []*searchRequestSpec {
	condensed := make([]*searchRequestSpec, len(queriesPerSpec))
	for spec, index := range specQueryIndex {
		queries := queriesPerSpec[index]
		if len(queries) == 1 {
			condensed[index] = &searchRequestSpec{
				base: &baseRequestSpec{
					Spec:  spec,
					Query: queries[0],
				},
			}
		} else {
			condensed[index] = &searchRequestSpec{
				base: &baseRequestSpec{
					Spec:  spec,
					Query: combineQueries(queries...),
				},
			}
		}
	}
	return condensed
}
