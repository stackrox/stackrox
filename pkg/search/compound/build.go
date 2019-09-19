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

func buildMultiSpec(q *v1.Query, specs []SearcherSpec) (*searchRequestSpec, error) {
	return treeBuilder(specs).walkSpecsRec(q)
}

// treeBuilder object stores the specs for each when walking the query.
type treeBuilder []SearcherSpec

func (tb treeBuilder) walkSpecsRec(q *v1.Query) (*searchRequestSpec, error) {
	if _, isDisjunction := q.GetQuery().(*v1.Query_Disjunction); isDisjunction {
		return tb.or(q.GetDisjunction())
	} else if _, isConjunction := q.GetQuery().(*v1.Query_Conjunction); isConjunction {
		return tb.and(q.GetConjunction())
	} else if _, isBool := q.GetQuery().(*v1.Query_BooleanQuery); isBool {
		return tb.boolean(q.GetBooleanQuery())
	} else if _, isBase := q.GetQuery().(*v1.Query_BaseQuery); isBase {
		return tb.base(q.GetBaseQuery())
	}
	return nil, fmt.Errorf("unrecognized query type: %T", q.GetQuery())
}

func (tb treeBuilder) or(q *v1.DisjunctionQuery) (*searchRequestSpec, error) {
	ret := make([]*searchRequestSpec, 0, len(q.GetQueries()))
	for _, dis := range q.GetQueries() {
		next, err := tb.walkSpecsRec(dis)
		if err != nil {
			return nil, err
		}
		ret = append(ret, next)
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
		ret = append(ret, next)
	}
	return &searchRequestSpec{
		and: ret,
	}, nil
}

func (tb treeBuilder) boolean(q *v1.BooleanQuery) (*searchRequestSpec, error) {
	must, err := tb.and(q.GetMust())
	if err != nil {
		return nil, err
	}

	mustNot, err := tb.or(q.GetMustNot())
	if err != nil {
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
		return tb.match(q.GetMatchFieldQuery())
	} else if _, isMatchLinkedField := q.GetQuery().(*v1.BaseQuery_MatchLinkedFieldsQuery); isMatchLinkedField {
		return tb.matchLinked(q.GetMatchLinkedFieldsQuery())
	}
	return nil, fmt.Errorf("cannot handle base query of type %T", q.GetQuery())
}

func (tb treeBuilder) docID(q *v1.DocIDQuery) *searchRequestSpec {
	return &searchRequestSpec{
		base: &baseRequestSpec{
			Spec: &tb[0],
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
	return &searchRequestSpec{
		base: &baseRequestSpec{
			Spec:  &tb[0],
			Query: search.MatchNoneQuery(),
		},
	}
}

func (tb treeBuilder) match(q *v1.MatchFieldQuery) (*searchRequestSpec, error) {
	// For other query types, we need to find the searcher that can handle it.
	for _, searcherSpec := range tb {
		if matchQueryMatchesOptions(q, searcherSpec.Options) {
			return &searchRequestSpec{
				base: &baseRequestSpec{
					Spec: &searcherSpec,
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
			}, nil
		}
	}
	return nil, errors.New("query had fields not handled by any searcher")
}

func (tb treeBuilder) matchLinked(q *v1.MatchLinkedFieldsQuery) (*searchRequestSpec, error) {
	// For other query types, we need to find the searcher that can handle it.
	for _, searcherSpec := range tb {
		if linkedMatchQueryMatchesOptions(q, searcherSpec.Options) {
			return &searchRequestSpec{
				base: &baseRequestSpec{
					Spec: &searcherSpec,
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
			}, nil
		}
	}
	return nil, errors.New("query had linked fields not handled by any searcher")
}

// Static helper functions.
///////////////////////////

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
