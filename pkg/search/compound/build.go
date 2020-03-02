package compound

import (
	"errors"
	"fmt"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// Build walks the query and maps every base query field to a single SearcherSpec. This way we know which parts of the
// query are concerned with which searcher.
func build(q *v1.Query, specs []SearcherSpec) (*searchRequestSpec, error) {
	if len(specs) == 0 {
		return nil, errors.New("searcher specs are required for building a search request")
	}
	// If we are only using a single searcher for some reason, then short circuit and just return a base node.
	if len(specs) == 1 {
		return buildSingleSpec(q, specs[0])
	}
	// Otherwise, we need to walk the tree and separate which query parts refer to which spec.
	return buildMultiSpec(q, specs)
}

func buildSingleSpec(q *v1.Query, spec SearcherSpec) (*searchRequestSpec, error) {
	return &searchRequestSpec{
		base: &baseRequestSpec{
			Spec:  &spec,
			Query: q,
		},
	}, nil
}

func buildDefaultSpec(specs []SearcherSpec, query *v1.Query) (*searchRequestSpec, error) {
	spec := getDefaultSpec(specs)
	if query == nil {
		query = search.EmptyQuery()
	}
	return &searchRequestSpec{
		base: &baseRequestSpec{
			Spec:  spec,
			Query: query,
		},
	}, nil
}

func buildMultiSpec(q *v1.Query, specs []SearcherSpec) (*searchRequestSpec, error) {
	spec, err := treeBuilder(specs).walkSpecsRec(q)
	if err != nil {
		return nil, err
	} else if spec == nil {
		return nil, errors.New("no matchable fields found")
	}
	return spec, nil
}

// treeBuilder object stores the specs for each when walking the query.
type treeBuilder []SearcherSpec

func (tb treeBuilder) walkSpecsRec(q *v1.Query) (*searchRequestSpec, error) {
	if q == nil || q.GetQuery() == nil {
		return buildDefaultSpec(tb, nil)
	}

	if _, isDisjunction := q.GetQuery().(*v1.Query_Disjunction); isDisjunction {
		return tb.or(q.GetDisjunction())
	} else if _, isConjunction := q.GetQuery().(*v1.Query_Conjunction); isConjunction {
		return tb.and(q.GetConjunction())
	} else if _, isBool := q.GetQuery().(*v1.Query_BooleanQuery); isBool {
		return tb.boolean(q.GetBooleanQuery())
	} else if _, isBase := q.GetQuery().(*v1.Query_BaseQuery); isBase {
		return tb.base(q.GetBaseQuery())
	}
	return buildDefaultSpec(tb, q)
}

func (tb treeBuilder) or(q *v1.DisjunctionQuery) (*searchRequestSpec, error) {
	ret := make([]*searchRequestSpec, 0, len(q.GetQueries()))
	for _, dis := range q.GetQueries() {
		next, err := tb.walkSpecsRec(dis)
		if err != nil {
			return nil, err
		}
		if next == nil {
			continue
		}
		ret = append(ret, next)
	}
	if len(ret) == 0 {
		return nil, nil
	}
	return &searchRequestSpec{
		or: ret,
	}, nil
}

func (tb treeBuilder) and(q *v1.ConjunctionQuery) (*searchRequestSpec, error) {
	ret := make([]*searchRequestSpec, 0, len(q.GetQueries()))
	for _, dis := range q.GetQueries() {
		next, err := tb.walkSpecsRec(dis)
		if err != nil {
			return nil, err
		}
		if next == nil {
			continue
		}
		ret = append(ret, next)
	}
	if len(ret) == 0 {
		return nil, nil
	}
	return &searchRequestSpec{
		and: ret,
	}, nil
}

func (tb treeBuilder) boolean(q *v1.BooleanQuery) (*searchRequestSpec, error) {
	must, err := tb.and(q.GetMust())
	if err != nil || must == nil {
		return nil, err
	}

	mustNot, err := tb.or(q.GetMustNot())
	if err != nil || mustNot == nil {
		return nil, err
	}

	return &searchRequestSpec{
		boolean: &booleanRequestSpec{
			must:    must,
			mustNot: mustNot,
		},
	}, nil
}

func (tb treeBuilder) base(q *v1.BaseQuery) (*searchRequestSpec, error) {
	// For DocId and MatchNone queries, we can always rely on the primary searcher.
	if _, isDocID := q.GetQuery().(*v1.BaseQuery_DocIdQuery); isDocID {
		return tb.docID(q.GetDocIdQuery()), nil
	} else if _, isMatchNone := q.GetQuery().(*v1.BaseQuery_MatchNoneQuery); isMatchNone {
		return tb.matchNone(q.GetMatchNoneQuery()), nil
	} else if _, isMatchField := q.GetQuery().(*v1.BaseQuery_MatchFieldQuery); isMatchField {
		return tb.match(q.GetMatchFieldQuery()), nil
	} else if _, isMatchLinkedField := q.GetQuery().(*v1.BaseQuery_MatchLinkedFieldsQuery); isMatchLinkedField {
		return tb.matchLinked(q.GetMatchLinkedFieldsQuery()), nil
	}
	return nil, fmt.Errorf("cannot handle base query of type %T", q.GetQuery())
}

func (tb treeBuilder) docID(q *v1.DocIDQuery) *searchRequestSpec {
	spec := getDefaultSpec(tb)
	return &searchRequestSpec{
		base: &baseRequestSpec{
			Spec: spec,
			Query: &v1.Query{
				Query: &v1.Query_BaseQuery{
					BaseQuery: &v1.BaseQuery{
						Query: &v1.BaseQuery_DocIdQuery{
							DocIdQuery: q,
						},
					},
				},
			},
		},
	}
}

func (tb treeBuilder) matchNone(q *v1.MatchNoneQuery) *searchRequestSpec {
	spec := getDefaultSpec(tb)
	return &searchRequestSpec{
		base: &baseRequestSpec{
			Spec:  spec,
			Query: search.MatchNoneQuery(),
		},
	}
}

func (tb treeBuilder) match(q *v1.MatchFieldQuery) *searchRequestSpec {
	spec, match := getMatchedSpec(tb, q)
	if !match {
		return nil
	}
	return &searchRequestSpec{
		base: &baseRequestSpec{
			Spec: spec,
			Query: &v1.Query{
				Query: &v1.Query_BaseQuery{
					BaseQuery: &v1.BaseQuery{
						Query: &v1.BaseQuery_MatchFieldQuery{
							MatchFieldQuery: q,
						},
					},
				},
			},
		},
	}
}

func (tb treeBuilder) matchLinked(q *v1.MatchLinkedFieldsQuery) *searchRequestSpec {
	// For other query types, we need to find the searcher that can handle it.
	spec, match := getLinkedMatchedSpec(tb, q)
	if !match {
		return nil
	}
	return &searchRequestSpec{
		base: &baseRequestSpec{
			Spec: spec,
			Query: &v1.Query{
				Query: &v1.Query_BaseQuery{
					BaseQuery: &v1.BaseQuery{
						Query: &v1.BaseQuery_MatchLinkedFieldsQuery{
							MatchLinkedFieldsQuery: q,
						},
					},
				},
			},
		},
	}
}

// Static helper functions.
///////////////////////////

func getDefaultSpec(specs []SearcherSpec) *SearcherSpec {
	var spec *SearcherSpec
	for _, considered := range specs {
		if considered.IsDefault {
			spec = &considered
			break
		}
	}
	if spec == nil {
		spec = &specs[0]
	}
	return spec
}

func getMatchedSpec(specs []SearcherSpec, query *v1.MatchFieldQuery) (*SearcherSpec, bool) {
	for _, searcherSpec := range specs {
		if matchQueryMatchesOptions(query, searcherSpec.Options) {
			return &searcherSpec, true
		}
	}
	return getDefaultSpec(specs), false
}

func getLinkedMatchedSpec(specs []SearcherSpec, query *v1.MatchLinkedFieldsQuery) (*SearcherSpec, bool) {
	for _, searcherSpec := range specs {
		if linkedMatchQueryMatchesOptions(query, searcherSpec.Options) {
			return &searcherSpec, true
		}
	}
	return getDefaultSpec(specs), false
}

// Base match query is chill if it matches a field in the option map.
func matchQueryMatchesOptions(q *v1.MatchFieldQuery, merp search.OptionsMap) bool {
	_, matches := merp.Get(q.GetField())
	return matches
}

// For a set of linked fields, they all need to match the same index, or the query doesn't make sense.
// Linked fields are for items in a single list as part of an object, like a key:value pair in a map. Having a key match
// in one object, and a value match in another is the same as failing the linked query in a single object.
func linkedMatchQueryMatchesOptions(q *v1.MatchLinkedFieldsQuery, merp search.OptionsMap) bool {
	for _, mq := range q.GetQuery() {
		if !matchQueryMatchesOptions(mq, merp) {
			return false
		}
	}
	return true
}
