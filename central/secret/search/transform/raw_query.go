package transform

import (
	"bitbucket.org/stack-rox/apollo/central/secret/store"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/search"
	"github.com/blevesearch/bleve"
)

// RawQueryWrapper wraps a RawQuery and provides functions for conversion.
type RawQueryWrapper struct {
	*v1.RawQuery
}

// ToSecrets returns the secrets that match the raw query.
func (r RawQueryWrapper) ToSecrets(storage store.Store, index bleve.Index) ([]*v1.Secret, error) {
	res, err := r.ToResults(index)
	if err != nil {
		return nil, err
	}
	return ResultsWrapper{Results: res}.ToSecrets(storage)
}

// ToRelationships returns the relationships that match the raw query.
func (r RawQueryWrapper) ToRelationships(storage store.Store, index bleve.Index) ([]*v1.SecretRelationship, error) {
	res, err := r.ToResults(index)
	if err != nil {
		return nil, err
	}
	return ResultsWrapper{Results: res}.ToRelationships(storage)
}

// ToResults returns the results that match the raw query.
func (r RawQueryWrapper) ToResults(index bleve.Index) ([]search.Result, error) {
	psr, err := r.ToParsedSearchRequest()
	if err != nil {
		return nil, err
	}
	return ParsedSearchRequestWrapper{ParsedSearchRequest: psr}.ToResults(index)
}

// ToParsedSearchRequest converts a raw query to a parsed search request.
func (r RawQueryWrapper) ToParsedSearchRequest() (*v1.ParsedSearchRequest, error) {
	if r.GetQuery() != "" {
		parser := &search.QueryParser{}
		return parser.ParseRawQuery(r.GetQuery())
	}
	return nil, nil
}
