package transform

import (
	"bitbucket.org/stack-rox/apollo/pkg/search"
	"bitbucket.org/stack-rox/apollo/pkg/search/blevesearch"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
)

// QueryWrapper wraps a query and provides functions for conversion.
type QueryWrapper struct {
	query.Query
}

// ToResults returns the results from the index that match the query.
func (q QueryWrapper) ToResults(index bleve.Index) ([]search.Result, error) {
	return blevesearch.RunQuery(q, index)
}
