package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/secret/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
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

func (d *datastoreImpl) GetSecret(ctx context.Context, id string) (*storage.Secret, bool, error) {
	secret, exists, err := d.storage.Get(ctx, id)
	if err != nil || !exists {
		return nil, false, err
	}

	if !secretSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).ForNamespaceScopedObject(secret).IsAllowed() {
		return nil, false, nil
	}

	return secret, true, nil
}

func (d *datastoreImpl) SearchSecrets(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	results, err := d.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	// TODO(ROX-29943): combine into 1 call to the database.
	ids := pkgSearch.ResultsToIDs(results)
	secrets, missingIndices, err := d.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	results = pkgSearch.RemoveMissingResults(results, missingIndices)
	return convertMany(secrets, results)
}

func (d *datastoreImpl) SearchListSecrets(ctx context.Context, request *v1.Query) ([]*storage.ListSecret, error) {
	secrets, err := d.SearchRawSecrets(ctx, request)
	if err != nil {
		return nil, err
	}

	listSecrets := make([]*storage.ListSecret, 0, len(secrets))
	for _, s := range secrets {
		listSecrets = append(listSecrets, convert.SecretToSecretList(s))
	}

	return listSecrets, nil
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

func (d *datastoreImpl) CountSecrets(ctx context.Context) (int, error) {
	if ok, err := secretSAC.ReadAllowed(ctx); err != nil {
		return 0, err
	} else if ok {
		return d.storage.Count(ctx, pkgSearch.EmptyQuery())
	}

	return d.Count(ctx, pkgSearch.EmptyQuery())
}

func (d *datastoreImpl) UpsertSecret(ctx context.Context, request *storage.Secret) error {
	if ok, err := secretSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return d.storage.Upsert(ctx, request)
}

func (d *datastoreImpl) RemoveSecret(ctx context.Context, id string) error {
	if ok, err := secretSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return d.storage.Delete(ctx, id)
}

func (d *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return d.storage.Search(ctx, q)
}

// Count returns the number of search results from the query
func (d *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return d.searcher.Count(ctx, q)
}

func convertMany(secrets []*storage.Secret, results []pkgSearch.Result) ([]*v1.SearchResult, error) {
	if len(results) != len(secrets) {
		return nil, errors.Errorf("expected %d search results, got %d objects", len(secrets), len(results))
	}

	outputResults := make([]*v1.SearchResult, len(secrets))
	for index, sar := range secrets {
		outputResults[index] = convertOne(sar, &results[index])
	}
	return outputResults, nil
}

func convertOne(secret *storage.Secret, result *pkgSearch.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_SECRETS,
		Id:             secret.GetId(),
		Name:           secret.GetName(),
		FieldToMatches: pkgSearch.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}
