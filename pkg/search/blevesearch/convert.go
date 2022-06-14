package blevesearch

import (
	"fmt"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

type queryConverter struct {
	category     v1.SearchCategory
	index        bleve.Index
	optionsMap   search.OptionsMap
	highlightCtx highlightContext
}

func newQueryConverter(category v1.SearchCategory, index bleve.Index, optionsMap search.OptionsMap) *queryConverter {
	return &queryConverter{
		category:     category,
		index:        index,
		optionsMap:   optionsMap,
		highlightCtx: make(highlightContext),
	}
}

func (c *queryConverter) convert(ctx bleveContext, q *v1.Query) (query.Query, highlightContext, error) {
	bleveQuery, err := c.convertHelper(ctx, q)
	return bleveQuery, c.highlightCtx, err
}

func (c *queryConverter) convertHelper(ctx bleveContext, q *v1.Query) (query.Query, error) {
	if q.GetQuery() == nil {
		return nil, nil
	}
	switch typedQ := q.GetQuery().(type) {
	case *v1.Query_BaseQuery:
		return c.baseQueryToBleve(ctx, typedQ.BaseQuery)
	case *v1.Query_Conjunction:
		return c.conjunctionQueryToBleve(ctx, typedQ.Conjunction)
	case *v1.Query_Disjunction:
		return c.disjunctionQueryToBleve(ctx, typedQ.Disjunction)
	case *v1.Query_BooleanQuery:
		return c.booleanQueryToBleve(ctx, typedQ.BooleanQuery)
	default:
		panic(fmt.Sprintf("Unhandled query type: %T", typedQ))
	}
}

func (c *queryConverter) baseQueryToBleve(ctx bleveContext, bq *v1.BaseQuery) (bleveQuery query.Query, err error) {
	if bq.GetQuery() == nil {
		return
	}

	switch bq := bq.GetQuery().(type) {
	case *v1.BaseQuery_MatchFieldQuery:
		return c.matchLinkedFieldsQueryToBleve(ctx, []*v1.MatchFieldQuery{bq.MatchFieldQuery})
	case *v1.BaseQuery_MatchLinkedFieldsQuery:
		return c.matchLinkedFieldsQueryToBleve(ctx, bq.MatchLinkedFieldsQuery.GetQuery())
	case *v1.BaseQuery_DocIdQuery:
		if len(bq.DocIdQuery.GetIds()) > 0 {
			bleveQuery = bleve.NewDocIDQuery(bq.DocIdQuery.GetIds())
		}
		return
	case *v1.BaseQuery_MatchNoneQuery:
		return bleve.NewMatchNoneQuery(), nil
	default:
		panic(fmt.Sprintf("Unhandled base query type: %T", bq))
	}
}

func (c *queryConverter) matchLinkedFieldsQueryToBleve(ctx bleveContext, mfqs []*v1.MatchFieldQuery) (bleveQuery query.Query, err error) {
	searchFieldsAndValues := make([]searchFieldAndValue, 0, len(mfqs))

	var category v1.SearchCategory
	var mustHighlight bool
	for _, mfq := range mfqs {
		if mfq.GetField() == "" || mfq.GetValue() == "" {
			continue
		}
		searchField, found := c.optionsMap.Get(mfq.GetField())
		if !found {
			continue
		}
		if category == v1.SearchCategory_SEARCH_UNSET {
			category = searchField.GetCategory()
		} else {
			if searchField.GetCategory() != category {
				return nil, fmt.Errorf("found multiple categories in query %+v ('%s' and '%s'), this is unsupported",
					mfqs, category, searchField.GetCategory())
			}
		}

		searchFieldsAndValues = append(searchFieldsAndValues, searchFieldAndValue{sf: searchField, value: mfq.GetValue(), highlight: mfq.GetHighlight()})
		if mfq.GetHighlight() {
			mustHighlight = true
		}
	}
	if len(searchFieldsAndValues) == 0 {
		return
	}
	var highlightCtx highlightContext
	if mustHighlight {
		highlightCtx = make(highlightContext)
	}
	bleveQuery, err = resolveMatchFieldQuery(ctx, c.index, c.category, searchFieldsAndValues, highlightCtx)
	if err != nil {
		return nil, err
	}
	if mustHighlight {
		c.highlightCtx.Merge(highlightCtx)
	}
	return
}

func (c *queryConverter) booleanQueryToBleve(ctx bleveContext, bq *v1.BooleanQuery) (query.Query, error) {
	cq, err := c.conjunctionQueryToBleve(ctx, bq.Must)
	if err != nil {
		return nil, err
	}
	dq, err := c.disjunctionQueryToBleve(ctx, bq.MustNot)
	if err != nil {
		return nil, err
	}
	boolQuery := bleve.NewBooleanQuery()
	boolQuery.AddMust(typeQuery(c.category))
	boolQuery.AddMust(cq.(*query.ConjunctionQuery).Conjuncts...)

	boolQuery.MustNot = dq
	return boolQuery, nil
}

func (c *queryConverter) conjunctionQueryToBleve(ctx bleveContext, cq *v1.ConjunctionQuery) (query.Query, error) {
	if len(cq.GetQueries()) == 0 {
		return nil, nil
	}
	bleveSubQueries, err := c.getBleveSubQueries(ctx, cq.GetQueries())
	if err != nil {
		return nil, err
	}
	if len(bleveSubQueries) == 0 {
		return nil, nil
	}
	return bleve.NewConjunctionQuery(bleveSubQueries...), nil
}

func (c *queryConverter) disjunctionQueryToBleve(ctx bleveContext, dq *v1.DisjunctionQuery) (query.Query, error) {
	if len(dq.GetQueries()) == 0 {
		return nil, nil
	}

	bleveSubQueries, err := c.getBleveSubQueries(ctx, dq.GetQueries())
	if err != nil {
		return nil, err
	}
	if len(bleveSubQueries) == 0 {
		return nil, nil
	}
	return bleve.NewDisjunctionQuery(bleveSubQueries...), nil
}

func (c *queryConverter) getBleveSubQueries(ctx bleveContext, qs []*v1.Query) ([]query.Query, error) {
	bleveSubQueries := make([]query.Query, 0, len(qs))
	for _, q := range qs {
		bleveSubQuery, err := c.convertHelper(ctx, q)
		if err != nil {
			return nil, err
		}
		if bleveSubQuery == nil {
			continue
		}
		bleveSubQueries = append(bleveSubQueries, bleveSubQuery)
	}
	return bleveSubQueries, nil
}
