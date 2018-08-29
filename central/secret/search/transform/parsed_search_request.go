package transform

import (
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	"github.com/stackrox/rox/central/secret/search/options"
	"github.com/stackrox/rox/central/secret/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

// ProtoQueryWrapper wraps a *v1.Query and provides functions for conversion.
type ProtoQueryWrapper struct {
	*v1.Query
}

// ToSearchResults converts the given parsed request to search results.
func (r ProtoQueryWrapper) ToSearchResults(storage store.Store, index bleve.Index) ([]*v1.SearchResult, error) {
	res, err := r.ToResults(index)
	if err != nil {
		return nil, err
	}
	return ResultsWrapper{Results: res}.ToSearchResults(storage)
}

// ToResults converts the given parsed request to the results of searching the bleve index.
func (r ProtoQueryWrapper) ToResults(index bleve.Index) ([]search.Result, error) {
	quer, err := r.toQuery(index)
	if err != nil {
		return nil, err
	}
	return QueryWrapper{Query: quer}.ToResults(index)
}

// ToQuery converts the given parsed request to a bleve query.
func (r ProtoQueryWrapper) toQuery(index bleve.Index) (query.Query, error) {
	sq, err := blevesearch.BuildQuery(index, v1.SearchCategory_SECRETS, r.Query, options.Map)
	if err != nil {
		return nil, err
	}
	return sq, nil
}
