package compound

import (
	"fmt"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/dackbox/keys/transformation"
	"github.com/stackrox/rox/pkg/search"
)

// build walks the query and maps every base query field to a single SearcherSpec. This way we know which parts of the
// query are concerned with which searcher.
func build(q *v1.Query, specs []SearcherSpec) (*searchRequestSpec, error) {
	if len(specs) == 0 {
		return nil, errors.New("searcher specs are required for building a search request")
	}
	// If we are only using a single searcher for some reason, then short circuit and just return a base node.
	if len(specs) == 1 {
		return buildSingleSpec(q, specs)
	}
	// Otherwise, we need to walk the tree and separate which query parts refer to which spec.
	return buildMultiSpec(q, specs)
}

func buildSingleSpec(q *v1.Query, specs []SearcherSpec) (*searchRequestSpec, error) {
	return &searchRequestSpec{
		base: &baseRequestSpec{
			Spec:  &specs[0],
			Query: q,
		},
	}, nil
}

func buildDefaultSpec(query *v1.Query, specs []SearcherSpec) (*searchRequestSpec, error) {
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
		return nil, nil
	}
	return spec, nil
}

// treeBuilder object stores the specs for each when walking the query.
type treeBuilder []SearcherSpec

func (tb treeBuilder) walkSpecsRec(q *v1.Query) (*searchRequestSpec, error) {
	if q == nil || q.GetQuery() == nil {
		return buildDefaultSpec(nil, tb)
	}

	if _, isDisjunction := q.GetQuery().(*v1.Query_Disjunction); isDisjunction {
		return tb.or(q.GetDisjunction().GetQueries())
	} else if _, isConjunction := q.GetQuery().(*v1.Query_Conjunction); isConjunction {
		return tb.and(q.GetConjunction().GetQueries())
	} else if _, isBool := q.GetQuery().(*v1.Query_BooleanQuery); isBool {
		return tb.boolean(q.GetBooleanQuery())
	} else if _, isBase := q.GetQuery().(*v1.Query_BaseQuery); isBase {
		return tb.base(q.GetBaseQuery())
	}
	return buildDefaultSpec(q, tb)
}

func (tb treeBuilder) or(queries []*v1.Query) (*searchRequestSpec, error) {
	ret := make([]*searchRequestSpec, 0, len(queries))
	for _, dis := range queries {
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

func (tb treeBuilder) and(queries []*v1.Query) (*searchRequestSpec, error) {
	ret := make([]*searchRequestSpec, 0, len(queries))
	for _, dis := range queries {
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
	must, err := tb.and(q.GetMust().GetQueries())
	if err != nil || must == nil {
		return nil, err
	}

	mustNot, err := tb.or(q.GetMustNot().GetQueries())
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
		return tb.matchLinked(q.GetMatchLinkedFieldsQuery())
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
	spec, match := getMatchedSpec(q, tb)
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

func (tb treeBuilder) matchLinked(q *v1.MatchLinkedFieldsQuery) (*searchRequestSpec, error) {
	spec := tb.matchLinkedSingle(q)
	if spec != nil {
		return spec, nil
	}
	return tb.matchLinkedSequence(q)
}

func (tb treeBuilder) matchLinkedSingle(q *v1.MatchLinkedFieldsQuery) *searchRequestSpec {
	// For other query types, we need to find the searcher that can handle it.
	spec, match := getLinkedMatchedSpec(q, tb)
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

func (tb treeBuilder) matchLinkedSequence(q *v1.MatchLinkedFieldsQuery) (*searchRequestSpec, error) {
	// We need to find the first searcher that handles one of the fields.
	offset := -1
	for i := len(tb) - 1; i >= 0; i-- {
		spec := tb[i]
		for _, matchField := range q.GetQuery() {
			if _, hasField := spec.Options.Get(matchField.GetField()); hasField {
				offset = i
				break
			}
		}
		if offset >= 0 {
			break
		}
	}
	if offset < 0 {
		return nil, errors.New("linked field search had fields that did not match")
	}

	// Transform linked to a conjunction.
	conjuncts := make([]*v1.Query, 0, len(q.GetQuery()))
	for _, matchField := range q.GetQuery() {
		conjuncts = append(conjuncts, &v1.Query{
			Query: &v1.Query_BaseQuery{
				BaseQuery: &v1.BaseQuery{
					Query: &v1.BaseQuery_MatchFieldQuery{
						MatchFieldQuery: matchField,
					},
				},
			},
		})
	}
	newQuery := search.ConjunctionQuery(conjuncts...)

	// Need to create a new SearcherSpec with that spec as the exit node.
	specsToUse, combinedOptions := recenterSpecRec(nil, tb[:offset+1])
	if len(specsToUse) == 0 {
		return nil, errors.New("specs not found despite field match")
	}

	// Now we need to build a new compound searcher with that as the root.
	return &searchRequestSpec{
		base: &baseRequestSpec{
			Spec: &SearcherSpec{
				IsDefault:      true,
				Searcher:       NewSearcher(specsToUse),
				Transformation: tb[offset].Transformation,
				Options:        combinedOptions,
			},
			Query: newQuery,
		},
	}, nil
}

func recenterSpecRec(prevTransform transformation.OneToMany, specs []SearcherSpec) ([]SearcherSpec, search.OptionsMap) {
	// If we have no spec, or no link, return
	if len(specs) == 0 {
		return nil, nil
	}
	spec := specs[len(specs)-1]

	retSpecs := []SearcherSpec{
		{
			IsDefault:      spec.IsDefault,
			Searcher:       spec.Searcher,
			Options:        spec.Options,
			Transformation: prevTransform,
		},
	}
	retOptions := spec.Options

	if spec.LinkToPrev != nil {
		transform := spec.LinkToPrev
		if prevTransform != nil {
			transform = transform.ThenMapEachToMany(prevTransform)
		}
		moreSpecs, moreOptions := recenterSpecRec(transform, specs[:len(specs)-1])
		if len(moreSpecs) > 0 {
			retSpecs = append(retSpecs, moreSpecs...)
			retOptions = search.CombineOptionsMaps(retOptions, moreOptions)
		}
	}

	return retSpecs, retOptions
}

// Static helper functions.
///////////////////////////

func getDefaultSpec(specs []SearcherSpec) *SearcherSpec {
	for _, spec := range specs {
		if spec.IsDefault {
			return &spec
		}
	}
	return &specs[0]
}

func getMatchedSpec(query *v1.MatchFieldQuery, specs []SearcherSpec) (*SearcherSpec, bool) {
	for _, spec := range specs {
		if matchQueryMatchesOptions(query, spec.Options) {
			return &spec, true
		}
	}
	return getDefaultSpec(specs), false
}

func getLinkedMatchedSpec(query *v1.MatchLinkedFieldsQuery, specs []SearcherSpec) (*SearcherSpec, bool) {
	for _, spec := range specs {
		if linkedMatchQueryMatchesOptions(query, spec.Options) {
			return &spec, true
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
