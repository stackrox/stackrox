package blevesearch

import (
	"fmt"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

type queryConverter struct {
	category     v1.SearchCategory
	index        bleve.Index
	optionsMap   map[search.FieldLabel]*v1.SearchField
	highlightCtx highlightContext
}

func newQueryConverter(category v1.SearchCategory, index bleve.Index, optionsMap map[search.FieldLabel]*v1.SearchField) *queryConverter {
	return &queryConverter{
		category:     category,
		index:        index,
		optionsMap:   optionsMap,
		highlightCtx: make(highlightContext),
	}
}

func (c *queryConverter) convert(q *v1.Query) (query.Query, highlightContext, error) {
	bleveQuery, err := c.convertHelper(q)
	return bleveQuery, c.highlightCtx, err
}

func (c *queryConverter) convertHelper(q *v1.Query) (query.Query, error) {
	if q.GetQuery() == nil {
		return nil, nil
	}
	switch typedQ := q.GetQuery().(type) {
	case *v1.Query_BaseQuery:
		return c.baseQueryToBleve(typedQ.BaseQuery)
	case *v1.Query_Conjunction:
		return c.conjunctionQueryToBleve(typedQ.Conjunction)
	case *v1.Query_Disjunction:
		return c.disjunctionQueryToBleve(typedQ.Disjunction)
	default:
		panic(fmt.Sprintf("Unhandled query type: %T", typedQ))
	}
}

func (c *queryConverter) baseQueryToBleve(bq *v1.BaseQuery) (bleveQuery query.Query, err error) {
	if bq.GetQuery() == nil {
		return
	}

	switch bq := bq.GetQuery().(type) {
	case *v1.BaseQuery_MatchFieldQuery:
		return c.matchLinkedFieldsQueryToBleve([]*v1.MatchFieldQuery{bq.MatchFieldQuery})
	case *v1.BaseQuery_MatchLinkedFieldsQuery:
		return c.matchLinkedFieldsQueryToBleve(bq.MatchLinkedFieldsQuery.GetQuery())
	case *v1.BaseQuery_DocIdQuery:
		if len(bq.DocIdQuery.GetIds()) > 0 {
			bleveQuery = bleve.NewDocIDQuery(bq.DocIdQuery.GetIds())
		}
		return
	default:
		panic(fmt.Sprintf("Unhandled base query type: %T", bq))
	}
}

func (c *queryConverter) matchLinkedFieldsQueryToBleve(mfqs []*v1.MatchFieldQuery) (bleveQuery query.Query, err error) {
	searchFieldsAndValues := make([]searchFieldAndValue, 0, len(mfqs))

	var category v1.SearchCategory
	var mustHighlight bool
	for _, mfq := range mfqs {
		if mfq.GetField() == "" || mfq.GetValue() == "" {
			continue
		}
		searchField, found := c.optionsMap[search.FieldLabel(mfq.GetField())]
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
	bleveQuery, err = resolveMatchFieldQuery(c.index, c.category, searchFieldsAndValues, highlightCtx)
	if err != nil {
		return nil, err
	}
	if mustHighlight {
		c.highlightCtx.Merge(highlightCtx)
	}
	return
}

func (c *queryConverter) conjunctionQueryToBleve(cq *v1.ConjunctionQuery) (query.Query, error) {
	if len(cq.GetQueries()) == 0 {
		return nil, nil
	}
	bleveSubQueries, err := c.getBleveSubQueries(cq.GetQueries())
	if err != nil {
		return nil, err
	}
	if len(bleveSubQueries) == 0 {
		return nil, nil
	}
	return bleve.NewConjunctionQuery(bleveSubQueries...), nil
}

func (c *queryConverter) disjunctionQueryToBleve(dq *v1.DisjunctionQuery) (query.Query, error) {
	if len(dq.GetQueries()) == 0 {
		return nil, nil
	}

	bleveSubQueries, err := c.getBleveSubQueries(dq.GetQueries())
	if err != nil {
		return nil, err
	}
	if len(bleveSubQueries) == 0 {
		return nil, nil
	}
	return bleve.NewDisjunctionQuery(bleveSubQueries...), nil
}

func (c *queryConverter) getBleveSubQueries(qs []*v1.Query) ([]query.Query, error) {
	bleveSubQueries := make([]query.Query, 0, len(qs))
	for _, q := range qs {
		bleveSubQuery, err := c.convertHelper(q)
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
