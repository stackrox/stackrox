package transform

import (
	"github.com/stackrox/rox/central/secret/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// ResultsWrapper provides functionality for transforming results from the index to useful data.
type ResultsWrapper struct {
	Results []search.Result
}

// ToSecrets returns the secrets from the db for the given search results.
func (r ResultsWrapper) ToSecrets(storage store.Store) ([]*v1.Secret, error) {
	ids := make([]string, len(r.Results), len(r.Results))
	for index, result := range r.Results {
		ids[index] = result.ID
	}
	return storage.GetSecretsBatch(ids)
}

// ToRelationships returns the relationships from the db for the given search results.
func (r ResultsWrapper) ToRelationships(storage store.Store) ([]*v1.SecretRelationship, error) {
	ids := make([]string, len(r.Results), len(r.Results))
	for index, result := range r.Results {
		ids[index] = result.ID
	}
	return storage.GetRelationshipBatch(ids)
}

// ToSearchResults returns the searchResults from the db for the given search results.
func (r ResultsWrapper) ToSearchResults(storage store.Store) ([]*v1.SearchResult, error) {
	sars, err := r.ToSecrets(storage)
	if err != nil {
		return nil, err
	}
	return convertMany(sars, r.Results), nil
}

func convertMany(secrets []*v1.Secret, results []search.Result) []*v1.SearchResult {
	outputResults := make([]*v1.SearchResult, len(secrets), len(secrets))
	for index, sar := range secrets {
		outputResults[index] = convertOne(sar, &results[index])
	}
	return outputResults
}

func convertOne(secret *v1.Secret, result *search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_SECRETS,
		Id:             secret.GetId(),
		Name:           secret.GetName(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}
