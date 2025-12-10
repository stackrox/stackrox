package datastore

import (
	"context"
	"strings"

	"github.com/stackrox/rox/central/cve/image/v2/datastore/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

type datastoreImpl struct {
	storage store.Store
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return ds.storage.Search(ctx, q)
}

func (ds *datastoreImpl) SearchImageCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	if q == nil {
		q = pkgSearch.EmptyQuery()
	}

	// Clone the query and add select fields for SearchResult construction
	clonedQuery := q.CloneVT()
	selectSelects := []*v1.QuerySelect{
		pkgSearch.NewQuerySelect(pkgSearch.CVE).Proto(),
	}
	clonedQuery.Selects = append(clonedQuery.GetSelects(), selectSelects...)

	results, err := ds.storage.Search(ctx, clonedQuery)
	if err != nil {
		return nil, err
	}
	searchTag := strings.ToLower(pkgSearch.CVE.String())
	for i := range results {
		if results[i].FieldValues != nil {
			if nameVal, ok := results[i].FieldValues[searchTag]; ok {
				results[i].Name = nameVal
			}
		}
	}

	return pkgSearch.ResultsToSearchResultProtos(results, &ImageCVESearchResultConverter{}), nil
}

func (ds *datastoreImpl) SearchRawImageCVEs(ctx context.Context, q *v1.Query) ([]*storage.ImageCVEV2, error) {
	var cves []*storage.ImageCVEV2
	err := ds.storage.GetByQueryFn(ctx, q, func(cve *storage.ImageCVEV2) error {
		cves = append(cves, cve)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return cves, nil
}

func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.storage.Count(ctx, q)
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.ImageCVEV2, bool, error) {
	cve, found, err := ds.storage.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}
	return cve, true, nil
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	found, err := ds.storage.Exists(ctx, id)
	if err != nil || !found {
		return false, err
	}
	return true, nil
}

func (ds *datastoreImpl) GetBatch(ctx context.Context, ids []string) ([]*storage.ImageCVEV2, error) {
	cves, _, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	return cves, nil
}

type ImageCVESearchResultConverter struct{}

func (c *ImageCVESearchResultConverter) BuildName(result *pkgSearch.Result) string {
	return result.Name
}

func (c *ImageCVESearchResultConverter) BuildLocation(result *pkgSearch.Result) string {
	return ""
}

func (c *ImageCVESearchResultConverter) GetCategory() v1.SearchCategory {
	return v1.SearchCategory_IMAGE_VULNERABILITIES_V2
}

func (c *ImageCVESearchResultConverter) GetScore(result *pkgSearch.Result) float64 {
	return result.Score
}
