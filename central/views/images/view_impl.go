package images

import (
	"context"

	"github.com/stackrox/rox/central/views/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

var (
	queryTimeout = env.PostgresVMStatementTimeout.DurationSetting()
)

type imageCoreViewImpl struct {
	schema *walker.Schema
	db     postgres.DB
}

func (v *imageCoreViewImpl) Get(ctx context.Context, query *v1.Query) ([]ImageCore, error) {
	if err := common.ValidateQuery(query); err != nil {
		return nil, err
	}

	var err error
	query, err = common.WithSACFilter(ctx, resources.Image, query)
	if err != nil {
		return nil, err
	}
	query = withSelectQuery(query)

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	var results []*imageResponse
	results, err = pgSearch.RunSelectRequestForSchema[imageResponse](queryCtx, v.db, v.schema, query)
	if err != nil {
		return nil, err
	}

	ret := make([]ImageCore, 0, len(results))
	for _, r := range results {
		ret = append(ret, r)
	}
	return ret, nil
}

func withSelectQuery(query *v1.Query) *v1.Query {
	cloned := query.CloneVT()
	var searchField search.FieldLabel
	if features.FlattenImageData.Enabled() {
		searchField = search.ImageID
	} else {
		searchField = search.ImageSHA
	}
	cloned.SetSelects([]*v1.QuerySelect{
		search.NewQuerySelect(searchField).Proto(),
	})

	if common.IsSortBySeverityCounts(cloned) {
		qgb := &v1.QueryGroupBy{}
		qgb.SetFields([]string{searchField.String()})
		cloned.SetGroupBy(qgb)
		cloned.SetSelects(append(cloned.GetSelects(),
			common.WithCountBySeverityAndFixabilityQuery(query, search.CVE).GetSelects()...,
		))
	}
	qgb := &v1.QueryGroupBy{}
	qgb.SetFields([]string{searchField.String()})
	cloned.SetGroupBy(qgb)

	// This is to minimize UI change and hide an implementation detail that the query groups images by their SHA.
	// Because of this, for a field that is not in images table, there can be multiple values of that field per SHA.
	// So in order to sort by that field, we need some kind of aggregate applied to it.
	for _, sortOption := range cloned.GetPagination().GetSortOptions() {
		if sortOption.GetField() == search.Severity.String() {
			sortOption.SetField(search.SeverityMax.String())
		}
		if sortOption.GetField() == search.CVSS.String() {
			sortOption.SetField(search.CVSSMax.String())
		}
		if sortOption.GetField() == search.NVDCVSS.String() {
			sortOption.SetField(search.NVDCVSSMax.String())
		}
		if sortOption.GetField() == search.OperatingSystem.String() {
			// Both 'Operating System' in CVE and 'Image OS' in an image containing that CVE have the same value.
			// Don't need an aggregate here since 'Image OS' is in images schema
			sortOption.SetField(search.ImageOS.String())
		}
	}

	return cloned
}
