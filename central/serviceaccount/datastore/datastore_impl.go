package datastore

import (
	"context"

	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/serviceaccount/internal/index"
	"github.com/stackrox/rox/central/serviceaccount/internal/store"
	"github.com/stackrox/rox/central/serviceaccount/search"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

const (
	batchSize = 1000
)

var (
	serviceAccountsSAC = sac.ForResource(resources.ServiceAccount)
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
	log.Info("[STARTUP] Indexing service accounts")
	var serviceAccounts []*storage.ServiceAccount
	var count int
	err := d.storage.Walk(ctx, func(sa *storage.ServiceAccount) error {
		serviceAccounts = append(serviceAccounts, sa)
		if len(serviceAccounts) == batchSize {
			if err := d.indexer.AddServiceAccounts(serviceAccounts); err != nil {
				return err
			}
			serviceAccounts = serviceAccounts[:0]
		}
		count++
		return nil
	})
	if err != nil {
		return err
	}
	if err := d.indexer.AddServiceAccounts(serviceAccounts); err != nil {
		return err
	}

	log.Infof("[STARTUP] Successfully indexed %d service accounts", count)
	return nil
}

func (d *datastoreImpl) GetServiceAccount(ctx context.Context, id string) (*storage.ServiceAccount, bool, error) {
	acc, found, err := d.storage.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}

	if ok, err := serviceAccountsSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).ForNamespaceScopedObject(acc).Allowed(); err != nil || !ok {
		return nil, false, err
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

	if err := d.storage.Upsert(ctx, request); err != nil {
		return err
	}
	return d.indexer.AddServiceAccount(request)
}

func (d *datastoreImpl) RemoveServiceAccount(ctx context.Context, id string) error {
	if ok, err := serviceAccountsSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if err := d.storage.Delete(ctx, id); err != nil {
		return err
	}
	return d.indexer.DeleteServiceAccount(id)
}

func (d *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return d.searcher.Search(ctx, q)
}

// Count returns the number of search results from the query
func (d *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return d.searcher.Count(ctx, q)
}
