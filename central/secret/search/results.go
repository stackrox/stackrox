package search

import (
	"bitbucket.org/stack-rox/apollo/central/secret/store"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/search"
)

// ResultsToSecrets returns the secrets from the db for the given search results.
func ResultsToSecrets(storage store.Store, results []search.Result) ([]*v1.Secret, error) {
	ids := make([]string, len(results), len(results))
	for index, result := range results {
		ids[index] = result.ID
	}
	return storage.GetSecretsBatch(ids)
}

// ResultsToRelationships returns the relationships from the db for the given search results.
func ResultsToRelationships(storage store.Store, results []search.Result) ([]*v1.SecretRelationship, error) {
	ids := make([]string, len(results), len(results))
	for index, result := range results {
		ids[index] = result.ID
	}
	return storage.GetRelationshipBatch(ids)
}

// ResultsToSearchResults returns the searchResults from the db for the given search results.
func ResultsToSearchResults(storage store.Store, results []search.Result) ([]*v1.SearchResult, error) {
	sars, err := ResultsToSecrets(storage, results)
	if err != nil {
		return nil, err
	}
	return convertMany(sars, results), nil
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
