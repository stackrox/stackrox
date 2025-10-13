package datastore

import (
	"context"

	"github.com/stackrox/rox/central/serviceaccount/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
)

const whenUnlimited = 100

var (
	serviceAccountsSAC = sac.ForResource(resources.ServiceAccount)
)

type datastoreImpl struct {
	storage store.Store
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
	serviceAccounts := make([]*storage.ServiceAccount, 0, paginated.GetLimit(q.GetPagination().GetLimit(), whenUnlimited))
	err := d.storage.GetByQueryFn(ctx, q, func(serviceAccount *storage.ServiceAccount) error {
		serviceAccounts = append(serviceAccounts, serviceAccount)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return serviceAccounts, nil
}

func (d *datastoreImpl) SearchServiceAccounts(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	serviceAccounts, results, err := d.searchServiceAccounts(ctx, q)
	if err != nil {
		return nil, err
	}

	return convertMany(serviceAccounts, results), nil
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
	return d.storage.Search(ctx, q)
}

// Count returns the number of search results from the query
func (d *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return d.storage.Count(ctx, q)
}

func (d *datastoreImpl) searchServiceAccounts(ctx context.Context, q *v1.Query) ([]*storage.ServiceAccount, []searchPkg.Result, error) {
	results, err := d.Search(ctx, q)
	if err != nil {
		return nil, nil, err
	}
	serviceAccounts, missingIndices, err := d.storage.GetMany(ctx, searchPkg.ResultsToIDs(results))
	if err != nil {
		return nil, nil, err
	}
	results = searchPkg.RemoveMissingResults(results, missingIndices)
	return serviceAccounts, results, nil
}

func convertMany(serviceAccounts []*storage.ServiceAccount, results []searchPkg.Result) []*v1.SearchResult {
	outputResults := make([]*v1.SearchResult, len(serviceAccounts))
	for index, sar := range serviceAccounts {
		outputResults[index] = convertServiceAccount(sar, &results[index])
	}
	return outputResults
}

func convertServiceAccount(sa *storage.ServiceAccount, result *searchPkg.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_SERVICE_ACCOUNTS,
		Id:             sa.GetId(),
		Name:           sa.GetName(),
		FieldToMatches: searchPkg.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}
