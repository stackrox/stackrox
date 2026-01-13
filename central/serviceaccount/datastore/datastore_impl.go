package datastore

import (
	"context"
	"strings"

	"github.com/stackrox/rox/central/serviceaccount/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
)

const whenUnlimited = 100

type datastoreImpl struct {
	storage store.Store
}

func (d *datastoreImpl) GetServiceAccount(ctx context.Context, id string) (*storage.ServiceAccount, bool, error) {
	acc, found, err := d.storage.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
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
	if q == nil {
		q = searchPkg.EmptyQuery()
	}
	qClone := q.CloneVT()

	// Add service account name field to select columns.
	qClone.Selects = append(qClone.GetSelects(), searchPkg.NewQuerySelect(searchPkg.ServiceAccountName).Proto())

	results, err := d.Search(ctx, qClone)
	if err != nil {
		return nil, err
	}

	// Extract name from FieldValues and populate Name in search results.
	searchTag := strings.ToLower(searchPkg.ServiceAccountName.String())
	for i := range results {
		if results[i].FieldValues != nil {
			if nameVal, ok := results[i].FieldValues[searchTag]; ok {
				results[i].Name = nameVal
			}
		}
	}

	return searchPkg.ResultsToSearchResultProtos(results, &serviceAccountSearchResultConverter{}), nil
}

func (d *datastoreImpl) UpsertServiceAccount(ctx context.Context, request *storage.ServiceAccount) error {
	return d.storage.Upsert(ctx, request)
}

func (d *datastoreImpl) RemoveServiceAccount(ctx context.Context, id string) error {
	return d.storage.Delete(ctx, id)
}

func (d *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return d.storage.Search(ctx, q)
}

// Count returns the number of search results from the query
func (d *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return d.storage.Count(ctx, q)
}

// serviceAccountSearchResultConverter implements searchPkg.SearchResultConverter for service accounts.
// It extracts name from Result.Name and does not set a location.
// Category is SERVICE_ACCOUNTS.

type serviceAccountSearchResultConverter struct{}

func (c *serviceAccountSearchResultConverter) BuildName(result *searchPkg.Result) string {
	return result.Name
}

func (c *serviceAccountSearchResultConverter) BuildLocation(result *searchPkg.Result) string {
	return ""
}

func (c *serviceAccountSearchResultConverter) GetCategory() v1.SearchCategory {
	return v1.SearchCategory_SERVICE_ACCOUNTS
}
