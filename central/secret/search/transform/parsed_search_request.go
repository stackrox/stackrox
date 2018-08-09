package transform

import (
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	"github.com/stackrox/rox/central/secret/index/mapping"
	"github.com/stackrox/rox/central/secret/search/options"
	"github.com/stackrox/rox/central/secret/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

// ParsedSearchRequestWrapper wraps a ParsedSearchRequest and provides functions for conversion.
type ParsedSearchRequestWrapper struct {
	*v1.ParsedSearchRequest
}

// ToSearchResults converts the given parsed request to search results.
func (r ParsedSearchRequestWrapper) ToSearchResults(storage store.Store, index bleve.Index) ([]*v1.SearchResult, error) {
	res, err := r.ToResults(index)
	if err != nil {
		return nil, err
	}
	return ResultsWrapper{Results: res}.ToSearchResults(storage)
}

// ToResults converts the given parsed request to the results of searching the bleve index.
func (r ParsedSearchRequestWrapper) ToResults(index bleve.Index) ([]search.Result, error) {
	quer, err := r.toQuery()
	if err != nil {
		return nil, err
	}
	return QueryWrapper{Query: quer}.ToResults(index)
}

// ToQuery converts the given parsed request to a bleve query.
func (r ParsedSearchRequestWrapper) toQuery() (query.Query, error) {
	// If the request is nil, just return all secrets.
	if r.ParsedSearchRequest == nil {
		sq := bleve.NewMatchQuery(mapping.IndexedType)
		sq.SetField("type")
		return sq, nil
	}

	// We search the indices for matches in both secret and relationship fields.
	sq, err := blevesearch.BuildQuery(mapping.IndexedType, r.ParsedSearchRequest, scopeToQuery, options.Map)
	if err != nil {
		return nil, err
	}
	return sq, nil
}

func scopeToQuery(scope *v1.Scope) query.Query {
	conjunctionQuery := bleve.NewConjunctionQuery()
	if scope.GetCluster() != "" {
		conjunctionQuery.AddQuery(blevesearch.NewMatchPhrasePrefixQuery("secret_and_relationship.relationship.cluster_relationship.name", scope.GetCluster()))
	}
	if scope.GetNamespace() != "" {
		conjunctionQuery.AddQuery(blevesearch.NewMatchPhrasePrefixQuery("secret_and_relationship.relationship.namespace_relationship.namespace", scope.GetNamespace()))
	}
	if len(conjunctionQuery.Conjuncts) == 0 {
		return bleve.NewMatchAllQuery()
	}
	return conjunctionQuery
}
