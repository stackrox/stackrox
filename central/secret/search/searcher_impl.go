package search

import (
	"context"

	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/central/secret/internal/index"
	"github.com/stackrox/stackrox/central/secret/internal/store"
	"github.com/stackrox/stackrox/central/secret/mappings"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/blevesearch"
	"github.com/stackrox/stackrox/pkg/search/paginated"
	"github.com/stackrox/stackrox/pkg/secret/convert"
)

var (
	defaultSortOption = &v1.QuerySortOption{
		Field: search.CreatedTime.String(),
	}

	secretSACSearchHelper         = sac.ForResource(resources.Secret).MustCreateSearchHelper(mappings.OptionsMap)
	secretSACPostgresSearchHelper = sac.ForResource(resources.Secret).MustCreatePgSearchHelper()
)

// searcherImpl provides an intermediary implementation layer for secrets
type searcherImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

// SearchSecrets returns the search results from indexed secrets for the query.
func (ds *searcherImpl) SearchSecrets(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	results, err := ds.getSearchResults(ctx, q)
	if err != nil {
		return nil, err
	}
	return ds.resultsToSearchResults(ctx, results)
}

// Search returns the raw search results from the query
func (ds *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return ds.getSearchResults(ctx, q)
}

// Count returns the number of search results from the query
func (ds *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.searcher.Count(ctx, q)
}

// SearchSecrets returns the secrets and relationships that match the query.
func (ds *searcherImpl) SearchListSecrets(ctx context.Context, q *v1.Query) ([]*storage.ListSecret, error) {
	results, err := ds.getSearchResults(ctx, q)
	if err != nil {
		return nil, err
	}
	secrets, _, err := ds.resultsToListSecrets(ctx, results)
	return secrets, err
}

// SearchRawSecrets retrieves secrets from the indexer and storage
func (ds *searcherImpl) SearchRawSecrets(ctx context.Context, q *v1.Query) ([]*storage.Secret, error) {
	return ds.searchSecrets(ctx, q)
}

func (ds *searcherImpl) getSearchResults(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return ds.searcher.Search(ctx, q)
}

// ToSecrets returns the secrets from the db for the given search results.
func (ds *searcherImpl) resultsToListSecrets(ctx context.Context, results []search.Result) ([]*storage.ListSecret, []int, error) {
	ids := search.ResultsToIDs(results)

	secrets, missingIndices, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	listSecrets := make([]*storage.ListSecret, 0, len(secrets))
	for _, s := range secrets {
		listSecrets = append(listSecrets, convert.SecretToSecretList(s))
	}
	return listSecrets, missingIndices, nil
}

// ToSearchResults returns the searchResults from the db for the given search results.
func (ds *searcherImpl) resultsToSearchResults(ctx context.Context, results []search.Result) ([]*v1.SearchResult, error) {
	secrets, missingIndices, err := ds.resultsToListSecrets(ctx, results)
	if err != nil {
		return nil, err
	}
	results = search.RemoveMissingResults(results, missingIndices)
	return convertMany(secrets, results), nil
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

// Format the search functionality of the indexer to be filtered (for sac) and paginated.
func formatSearcher(unsafeSearcher blevesearch.UnsafeSearcher) search.Searcher {
	var filteredSearcher search.Searcher
	if features.PostgresDatastore.Enabled() {
		// Make the UnsafeSearcher safe.
		filteredSearcher = secretSACPostgresSearchHelper.FilteredSearcher(unsafeSearcher)
	} else {
		filteredSearcher = secretSACSearchHelper.FilteredSearcher(unsafeSearcher) // Make the UnsafeSearcher safe.
	}
	paginatedSearcher := paginated.Paginated(filteredSearcher)
	defaultSortedSearcher := paginated.WithDefaultSortOption(paginatedSearcher, defaultSortOption)
	return defaultSortedSearcher
}

func (ds *searcherImpl) searchSecrets(ctx context.Context, q *v1.Query) ([]*storage.Secret, error) {
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	ids := search.ResultsToIDs(results)
	secrets, _, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	return secrets, nil
}
