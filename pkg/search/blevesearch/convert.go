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

	switch typedBQ := bq.GetQuery().(type) {
	case *v1.BaseQuery_StringQuery:
		if typedBQ.StringQuery.GetQuery() != "" {
			bleveQuery = NewMatchPhrasePrefixQuery("", typedBQ.StringQuery.GetQuery())
		}
		return
	case *v1.BaseQuery_MatchFieldQuery:
		return c.matchFieldQueryToBleve(typedBQ.MatchFieldQuery)
	default:
		panic(fmt.Sprintf("Unhandled base query type: %T", typedBQ))
	}
}

func (c *queryConverter) matchFieldQueryToBleve(mq *v1.MatchFieldQuery) (bleveQuery query.Query, err error) {
	if mq.GetField() == "" || mq.GetValue() == "" {
		return
	}
	searchField, found := c.optionsMap[search.FieldLabel(mq.GetField())]
	if !found {
		return
	}
	var highlightCtx highlightContext
	if mq.GetHighlight() {
		highlightCtx = make(highlightContext)
	}
	bleveQuery, err = resolveMatchFieldQuery(c.index, c.category, searchField, mq.GetValue(), highlightCtx)
	if err != nil {
		return nil, err
	}
	if mq.GetHighlight() {
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
