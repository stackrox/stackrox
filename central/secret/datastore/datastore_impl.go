package datastore

import (
	"context"

	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/secret/internal/index"
	"github.com/stackrox/rox/central/secret/internal/store"
	"github.com/stackrox/rox/central/secret/search"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/debug"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

var (
	secretSAC = sac.ForResource(resources.Secret)
)

type datastoreImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

func (d *datastoreImpl) buildIndex(ctx context.Context) error {
	if features.PostgresDatastore.Enabled() {
		return nil
	}
	defer debug.FreeOSMemory()
	log.Info("[STARTUP] Indexing secrets")

	var secrets []*storage.Secret
	err := d.storage.Walk(ctx, func(secret *storage.Secret) error {
		secrets = append(secrets, secret)
		return nil
	})
	if err != nil {
		return err
	}
	if err := d.indexer.AddSecrets(secrets); err != nil {
		return err
	}
	log.Info("[STARTUP] Successfully indexed secrets")
	return nil
}

func (d *datastoreImpl) GetSecret(ctx context.Context, id string) (*storage.Secret, bool, error) {
	secret, exists, err := d.storage.Get(ctx, id)
	if err != nil || !exists {
		return nil, false, err
	}

	if ok, err := secretSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).ForNamespaceScopedObject(secret).Allowed(); err != nil || !ok {
		return nil, false, err
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

	if err := d.storage.Upsert(ctx, request); err != nil {
		return err
	}
	return d.indexer.AddSecret(request)
}

func (d *datastoreImpl) RemoveSecret(ctx context.Context, id string) error {
	if ok, err := secretSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if err := d.storage.Delete(ctx, id); err != nil {
		return err
	}
	return d.indexer.DeleteSecret(id)
}

func (d *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return d.searcher.Search(ctx, q)
}

// Count returns the number of search results from the query
func (d *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return d.searcher.Count(ctx, q)
}
