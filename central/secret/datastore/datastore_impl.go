package datastore

import (
	"context"

	"github.com/stackrox/rox/central/secret/internal/store"
	"github.com/stackrox/rox/central/secret/search"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

var (
	secretSAC = sac.ForResource(resources.Secret)
)

type datastoreImpl struct {
	storage  store.Store
	searcher search.Searcher
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
	return d.searcher.SearchSecrets(ctx, q)
}

func (d *datastoreImpl) SearchListSecrets(ctx context.Context, request *v1.Query) ([]*storage.ListSecret, error) {
	return d.searcher.SearchListSecrets(ctx, request)
}

func (d *datastoreImpl) SearchRawSecrets(ctx context.Context, request *v1.Query) ([]*storage.Secret, error) {
	return d.searcher.SearchRawSecrets(ctx, request)
}

func (d *datastoreImpl) CountSecrets(ctx context.Context) (int, error) {
	if ok, err := secretSAC.ReadAllowed(ctx); err != nil {
		return 0, err
	} else if ok {
		return d.storage.Count(ctx)
	}

	return d.Count(ctx, searchPkg.EmptyQuery())
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

func (d *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return d.searcher.Search(ctx, q)
}

// Count returns the number of search results from the query
func (d *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return d.searcher.Count(ctx, q)
}
