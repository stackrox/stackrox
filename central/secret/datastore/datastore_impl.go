package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/secret/internal/store"
	pgStore "github.com/stackrox/rox/central/secret/internal/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/secret/convert"
)

var (
	secretSAC = sac.ForResource(resources.Secret)
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
	// TODO(ROX-29943): remove 2 pass database queries
	results, err := d.storage.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	secrets, missingIndices, err := d.resultsToListSecrets(ctx, results)
	if err != nil {
		return nil, err
	}
	results = pkgSearch.RemoveMissingResults(results, missingIndices)
	return convertMany(secrets, results)
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

func convertMany(secrets []*storage.ListSecret, results []pkgSearch.Result) ([]*v1.SearchResult, error) {
	if len(secrets) != len(results) {
		return nil, errors.Errorf("expected %d secrets but got %d", len(results), len(secrets))
	}

	outputResults := make([]*v1.SearchResult, len(secrets))
	for index, sar := range secrets {
		outputResults[index] = convertOne(sar, &results[index])
	}
	return outputResults, nil
}

func convertOne(secret *storage.ListSecret, result *pkgSearch.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_SECRETS,
		Id:             secret.GetId(),
		Name:           secret.GetName(),
		FieldToMatches: pkgSearch.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}
