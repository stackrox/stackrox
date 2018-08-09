package search

import (
	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/secret/search/transform"
	"github.com/stackrox/rox/central/secret/store"
	"github.com/stackrox/rox/generated/api/v1"
)

// searcherImpl provides an intermediary implementation layer for AlertStorage.
type searcherImpl struct {
	storage store.Store
	index   bleve.Index
}

// SearchSecrets returns the search results from indexed secrets for the query.
func (ds *searcherImpl) SearchSecrets(rawQuery *v1.RawQuery) ([]*v1.SearchResult, error) {
	results, err := transform.RawQueryWrapper{RawQuery: rawQuery}.ToResults(ds.index)
	if err != nil {
		return nil, err
	}
	return transform.ResultsWrapper{Results: results}.ToSearchResults(ds.storage)
}

// SearchSecrets returns the secrets and relationships that match the query.
func (ds *searcherImpl) SearchRawSecrets(rawQuery *v1.RawQuery) ([]*v1.SecretAndRelationship, error) {
	secrets, err := transform.RawQueryWrapper{RawQuery: rawQuery}.ToSecrets(ds.storage, ds.index)
	if err != nil {
		return nil, err
	}

	relationships, err := transform.RawQueryWrapper{RawQuery: rawQuery}.ToRelationships(ds.storage, ds.index)
	if err != nil {
		return nil, err
	}

	var sars []*v1.SecretAndRelationship
	for index, secret := range secrets {
		sar := &v1.SecretAndRelationship{
			Secret:       secret,
			Relationship: relationships[index],
		}
		sars = append(sars, sar)
	}
	return sars, nil
}
