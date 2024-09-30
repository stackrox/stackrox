package images

import (
	"context"

	"github.com/stackrox/rox/central/views/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
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

	var results []*imageResponse
	results, err = pgSearch.RunSelectRequestForSchema[imageResponse](ctx, v.db, v.schema, query)
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
	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{search.ImageSHA.String()},
	}

	if common.IsSortBySeverityCounts(cloned) {
		cloned.Selects = append(cloned.Selects,
			common.WithCountBySeverityAndFixabilityQuery(query, search.CVE).Selects...,
		)
	}

	return cloned
}
