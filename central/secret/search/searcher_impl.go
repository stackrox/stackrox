package search

import (
	"context"

	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/secret/internal/index"
	"github.com/stackrox/rox/central/secret/internal/store"
	"github.com/stackrox/rox/central/secret/mappings"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
)

var (
	secretSACSearchHelper = sac.ForResource(resources.Secret).MustCreateSearchHelper(mappings.OptionsMap, sac.ClusterIDAndNamespaceFields)
)

// searcherImpl provides an intermediary implementation layer focentral/serviceaccount/search/searcher_impl.gor AlertStorage.
type searcherImpl struct {
	storage store.Store
	indexer index.Indexer
}

// SearchSecrets returns the search results from indexed secrets for the query.
func (ds *searcherImpl) SearchSecrets(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	results, err := ds.getSearchResults(ctx, q)
	if err != nil {
		return nil, err
	}
	return ds.resultsToSearchResults(results)
}

// Search returns the raw search results from the query
func (ds *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return ds.getSearchResults(ctx, q)
}

// SearchSecrets returns the secrets and relationships that match the query.
func (ds *searcherImpl) SearchListSecrets(ctx context.Context, q *v1.Query) ([]*storage.ListSecret, error) {
	results, err := ds.getSearchResults(ctx, q)
	if err != nil {
		return nil, err
	}
	return ds.resultsToListSecrets(results)
}

func (ds *searcherImpl) getSearchResults(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return secretSACSearchHelper.Apply(ds.indexer.Search)(ctx, q)
}

// ToSecrets returns the secrets from the db for the given search results.
func (ds *searcherImpl) resultsToListSecrets(results []search.Result) ([]*storage.ListSecret, error) {
	ids := make([]string, len(results))
	for index, result := range results {
		ids[index] = result.ID
	}
	return ds.storage.ListSecrets(ids)
}

// ToSearchResults returns the searchResults from the db for the given search results.
func (ds *searcherImpl) resultsToSearchResults(results []search.Result) ([]*v1.SearchResult, error) {
	sars, err := ds.resultsToListSecrets(results)
	if err != nil {
		return nil, err
	}
	return convertMany(sars, results), nil
}

func convertMany(secrets []*storage.ListSecret, results []search.Result) []*v1.SearchResult {
	outputResults := make([]*v1.SearchResult, len(secrets))
	for index, sar := range secrets {
		outputResults[index] = convertOne(sar, &results[index])
	}
	return outputResults
}

func convertOne(secret *storage.ListSecret, result *search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_SECRETS,
		Id:             secret.GetId(),
		Name:           secret.GetName(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}
