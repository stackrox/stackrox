package datastore

import (
	"context"
	"strings"

	"github.com/stackrox/rox/central/secret/internal/store"
	pgStore "github.com/stackrox/rox/central/secret/internal/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/secret/convert"
)

type datastoreImpl struct {
	storage store.Store
}

func newPostgres(db postgres.DB) DataStore {
	dbStore := pgStore.New(db)
	return &datastoreImpl{
		storage: dbStore,
	}
}

func (d *datastoreImpl) GetSecret(ctx context.Context, id string) (*storage.Secret, bool, error) {
	secret, exists, err := d.storage.Get(ctx, id)
	if err != nil || !exists {
		return nil, false, err
	}

	return secret, true, nil
}

func (d *datastoreImpl) SearchSecrets(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	if q == nil {
		q = pkgSearch.EmptyQuery()
	} else {
		q = q.CloneVT()
	}

	// Add name field to select columns
	q.Selects = append(q.GetSelects(), pkgSearch.NewQuerySelect(pkgSearch.SecretName).Proto())

	results, err := d.storage.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	// Extract name from FieldValues and populate Name in search results
	searchTag := strings.ToLower(pkgSearch.SecretName.String())
	for i := range results {
		if results[i].FieldValues != nil {
			if nameVal, ok := results[i].FieldValues[searchTag]; ok {
				results[i].Name = nameVal
			}
		}
	}

	return pkgSearch.ResultsToSearchResultProtos(results, &SecretSearchResultConverter{}), nil
}

func (d *datastoreImpl) SearchListSecrets(ctx context.Context, request *v1.Query) ([]*storage.ListSecret, error) {
	results, err := d.Search(ctx, request)
	if err != nil {
		return nil, err
	}
	secrets, _, err := d.resultsToListSecrets(ctx, results)
	return secrets, err
}

func (d *datastoreImpl) SearchRawSecrets(ctx context.Context, request *v1.Query) ([]*storage.Secret, error) {
	var secrets []*storage.Secret
	err := d.storage.GetByQueryFn(ctx, request, func(secret *storage.Secret) error {
		secrets = append(secrets, secret)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return secrets, nil
}

func (d *datastoreImpl) UpsertSecret(ctx context.Context, request *storage.Secret) error {
	return d.storage.Upsert(ctx, request)
}

func (d *datastoreImpl) RemoveSecret(ctx context.Context, id string) error {
	return d.storage.Delete(ctx, id)
}

func (d *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return d.storage.Search(ctx, q)
}

// Count returns the number of search results from the query
func (d *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return d.storage.Count(ctx, q)
}

// ToSecrets returns the secrets from the db for the given search results.
func (d *datastoreImpl) resultsToListSecrets(ctx context.Context, results []pkgSearch.Result) ([]*storage.ListSecret, []int, error) {
	ids := pkgSearch.ResultsToIDs(results)

	secrets, missingIndices, err := d.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	listSecrets := make([]*storage.ListSecret, 0, len(secrets))
	for _, s := range secrets {
		listSecrets = append(listSecrets, convert.SecretToSecretList(s))
	}
	return listSecrets, missingIndices, nil
}

type SecretSearchResultConverter struct{}

func (c *SecretSearchResultConverter) BuildName(result *pkgSearch.Result) string {
	return result.Name
}

func (c *SecretSearchResultConverter) BuildLocation(result *pkgSearch.Result) string {
	// Secrets do not have a location
	return ""
}

func (c *SecretSearchResultConverter) GetCategory() v1.SearchCategory {
	return v1.SearchCategory_SECRETS
}

func (c *SecretSearchResultConverter) GetScore(result *pkgSearch.Result) float64 {
	return result.Score
}
