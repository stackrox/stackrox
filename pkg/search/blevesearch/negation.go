package blevesearch

import (
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/query"
	index "github.com/blevesearch/bleve_index_api"
)

// NegationQuery is the negated form of the passed query
type NegationQuery struct {
	typeQuery query.Query
	query     query.Query
	required  bool
}

// NewNegationQuery creates a new NegationQuery for finding a negated version of the query
func NewNegationQuery(typeQuery, subQuery query.Query, required bool) *NegationQuery {
	return &NegationQuery{
		typeQuery: typeQuery,
		query:     subQuery,
		required:  required,
	}
}

// Searcher returns the negation searcher
func (q *NegationQuery) Searcher(i index.IndexReader, m mapping.IndexMapping, options search.SearcherOptions) (search.Searcher, error) {
	typeSearcher, err := q.typeQuery.Searcher(i, m, options)
	if err != nil {
		return nil, err
	}
	qSearcher, err := q.query.Searcher(i, m, options)
	if err != nil {
		return nil, err
	}
	return NewNegationSearcher(i, typeSearcher, qSearcher, options, q.required)
}
