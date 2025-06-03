package images

import (
	"context"

	"github.com/stackrox/rox/central/views/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/env"
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
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.ImageSHA).Proto(),
	}

	if common.IsSortBySeverityCounts(cloned) {
		cloned.GroupBy = &v1.QueryGroupBy{
			Fields: []string{search.ImageSHA.String()},
		}
		cloned.Selects = append(cloned.Selects,
			common.WithCountBySeverityAndFixabilityQuery(query, search.CVE).Selects...,
		)
	}
	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{search.ImageSHA.String()},
	}

	// This is to minimize UI change and hide an implementation detail that the query groups images by their SHA.
	// Because of this, for a field that is not in images table, there can be multiple values of that field per SHA.
	// So in order to sort by that field, we need some kind of aggregate applied to it.
	for _, sortOption := range cloned.GetPagination().GetSortOptions() {
		if sortOption.Field == search.Severity.String() {
			sortOption.Field = search.SeverityMax.String()
		}
		if sortOption.Field == search.CVSS.String() {
			sortOption.Field = search.CVSSMax.String()
		}
		if sortOption.Field == search.NVDCVSS.String() {
			sortOption.Field = search.NVDCVSSMax.String()
		}
		if sortOption.Field == search.OperatingSystem.String() {
			// Both 'Operating System' in CVE and 'Image OS' in an image containing that CVE have the same value.
			// Don't need an aggregate here since 'Image OS' is in images schema
			sortOption.Field = search.ImageOS.String()
		}
	}

	return cloned
}
