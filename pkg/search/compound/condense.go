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
	condensed, err := condenseList(req.or, search.NewDisjunctionQuery)
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
	condensed, err := condenseList(req.and, search.NewConjunctionQuery)
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
	bySpec := make(map[*SearcherSpec][]*v1.Query)
	for _, child := range children {
		condensed, err := condense(child)
		if err != nil {
			return nil, err
		}
		if condensed.base == nil {
			ret = append(ret, condensed)
		} else {
			bySpec[condensed.base.Spec] = append(bySpec[condensed.base.Spec], condensed.base.Query)
		}
	}
	if len(bySpec) > 0 {
		ret = append(ret, condenseMap(bySpec, combineQueries)...)
	}
	return ret, nil
}

func condenseMap(condensable map[*SearcherSpec][]*v1.Query, combineQueries func(q ...*v1.Query) *v1.Query) []*searchRequestSpec {
	condensed := make([]*searchRequestSpec, 0, len(condensable))
	for spec, queries := range condensable {
		if len(queries) == 1 {
			condensed = append(condensed, &searchRequestSpec{
				base: &baseRequestSpec{
					Spec:  spec,
					Query: queries[0],
				},
			})
		} else {
			condensed = append(condensed, &searchRequestSpec{
				base: &baseRequestSpec{
					Spec:  spec,
					Query: combineQueries(queries...),
				},
			})
		}
	}
	return condensed
}
