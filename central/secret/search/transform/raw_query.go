package transform

import (
	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/secret/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
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
	psr, err := r.ToProtoQuery()
	if err != nil {
		return nil, err
	}
	return ProtoQueryWrapper{Query: psr}.ToResults(index)
}

// ToProtoQuery converts a raw query to a proto query.
func (r RawQueryWrapper) ToProtoQuery() (*v1.Query, error) {
	if r.GetQuery() != "" {
		return search.ParseRawQuery(r.GetQuery())
	}
	return nil, nil
}
