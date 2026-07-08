package datastore

import (
	"context"
	"strings"

	"github.com/stackrox/rox/central/cve/image/v2/datastore/store"
	imagev2common "github.com/stackrox/rox/central/imagev2/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
)

type datastoreImpl struct {
	storage store.Store
	db      postgres.DB
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	q = imagev2common.WithRowsFromImageV2Only(q)
	return ds.storage.Search(ctx, q)
}

func (ds *datastoreImpl) SearchImageCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	if q == nil {
		q = pkgSearch.EmptyQuery()
	}
	q = imagev2common.WithRowsFromImageV2Only(q)

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
	q = imagev2common.WithRowsFromImageV2Only(q)
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
	q = imagev2common.WithRowsFromImageV2Only(q)
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

type digestCVECountView struct {
	ImageID  *string `db:"cve_image_id"`
	CVECount int     `db:"cve_id_count"`
}

// GetDigestsWithMostV1CVEs returns up to `limit` image digests that have V1 CVE rows
// (imageid set, imageidv2 NULL), ordered by CVE count descending.
func (ds *datastoreImpl) GetDigestsWithMostV1CVEs(ctx context.Context, limit int) ([]string, error) {
	q := pkgSearch.NewQueryBuilder().
		AddNullField(pkgSearch.CVEImageIDV2).
		ProtoQuery()
	q.Selects = []*v1.QuerySelect{
		pkgSearch.NewQuerySelect(pkgSearch.CVEImageID).Proto(),
		pkgSearch.NewQuerySelect(pkgSearch.CVEID).AggrFunc(aggregatefunc.Count).Proto(),
	}
	q.GroupBy = &v1.QueryGroupBy{
		Fields: []string{pkgSearch.CVEImageID.String()},
	}
	q.Pagination = pkgSearch.NewPagination().
		Limit(int32(limit)).
		AddSortOption(pkgSearch.NewSortOption(pkgSearch.CVEID).AggregateBy(aggregatefunc.Count, false).Reversed(true)).
		Proto()

	var digests []string
	err := pgSearch.RunSelectRequestForSchemaFn(ctx, ds.db, pkgSchema.ImageCvesV2Schema, q, func(row *digestCVECountView) error {
		if row.ImageID != nil {
			digests = append(digests, *row.ImageID)
		}
		return nil
	})
	return digests, err
}

func (ds *datastoreImpl) GetV1CVEsByDigests(ctx context.Context, digests []string) ([]*CVETimestampsView, error) {
	if len(digests) == 0 {
		return nil, nil
	}
	q := pkgSearch.NewQueryBuilder().
		AddExactMatches(pkgSearch.CVEImageID, digests...).
		AddNullField(pkgSearch.CVEImageIDV2).
		ProtoQuery()
	q.Selects = cveTimestampsViewSelects()

	var results []*CVETimestampsView
	err := pgSearch.RunSelectRequestForSchemaFn(ctx, ds.db, pkgSchema.ImageCvesV2Schema, q, func(row *CVETimestampsView) error {
		results = append(results, row)
		return nil
	})
	return results, err
}

func (ds *datastoreImpl) GetV2CVEsByImageIDs(ctx context.Context, imageIDs []string) ([]*CVETimestampsView, error) {
	if len(imageIDs) == 0 {
		return nil, nil
	}
	q := pkgSearch.NewQueryBuilder().
		AddExactMatches(pkgSearch.CVEImageIDV2, imageIDs...).
		ProtoQuery()
	q.Selects = cveTimestampsViewSelects()

	var results []*CVETimestampsView
	err := pgSearch.RunSelectRequestForSchemaFn(ctx, ds.db, pkgSchema.ImageCvesV2Schema, q, func(row *CVETimestampsView) error {
		results = append(results, row)
		return nil
	})
	return results, err
}

func cveTimestampsViewSelects() []*v1.QuerySelect {
	return []*v1.QuerySelect{
		pkgSearch.NewQuerySelect(pkgSearch.CVEID).Proto(),
		pkgSearch.NewQuerySelect(pkgSearch.CVEImageID).Proto(),
		pkgSearch.NewQuerySelect(pkgSearch.CVEImageIDV2).Proto(),
		pkgSearch.NewQuerySelect(pkgSearch.CVE).Proto(),
		pkgSearch.NewQuerySelect(pkgSearch.Fixable).Proto(),
		pkgSearch.NewQuerySelect(pkgSearch.CVECreatedTime).Proto(),
		pkgSearch.NewQuerySelect(pkgSearch.FirstImageOccurrenceTimestamp).Proto(),
		pkgSearch.NewQuerySelect(pkgSearch.CVEFixAvailable).Proto(),
	}
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
