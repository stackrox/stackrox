package datastore

import (
	"context"

	"github.com/stackrox/rox/central/serviceaccount/internal/store"
	"github.com/stackrox/rox/central/serviceaccount/search"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

var (
	serviceAccountsSAC = sac.ForResource(resources.ServiceAccount)
)

type datastoreImpl struct {
	storage  store.Store
	searcher search.Searcher
}

func (d *datastoreImpl) GetServiceAccount(ctx context.Context, id string) (*storage.ServiceAccount, bool, error) {
	acc, found, err := d.storage.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}

	if !serviceAccountsSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).ForNamespaceScopedObject(acc).IsAllowed() {
		return nil, false, nil
	}

	return acc, true, nil
}

func (d *datastoreImpl) SearchRawServiceAccounts(ctx context.Context, q *v1.Query) ([]*storage.ServiceAccount, error) {
	return d.searcher.SearchRawServiceAccounts(ctx, q)
}

func (d *datastoreImpl) SearchServiceAccounts(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return d.searcher.SearchServiceAccounts(ctx, q)
}

func (d *datastoreImpl) UpsertServiceAccount(ctx context.Context, request *storage.ServiceAccount) error {
	if ok, err := serviceAccountsSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return d.storage.Upsert(ctx, request)
}

func (d *datastoreImpl) RemoveServiceAccount(ctx context.Context, id string) error {
	if ok, err := serviceAccountsSAC.WriteAllowed(ctx); err != nil {
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
