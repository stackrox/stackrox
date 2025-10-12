package imagecomponentflat

import (
	"context"
	"sort"

	"github.com/stackrox/rox/central/views/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
)

var (
	queryTimeout = env.PostgresVMStatementTimeout.DurationSetting()
)

type imageComponentFlatViewImpl struct {
	schema *walker.Schema
	db     postgres.DB
}

func (v *imageComponentFlatViewImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	if err := common.ValidateQuery(q); err != nil {
		return 0, err
	}

	var err error
	q, err = common.WithSACFilter(ctx, resources.Image, q)
	if err != nil {
		return 0, err
	}

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	clonedQ := q.CloneVT()
	clonedQ.Pagination = nil

	// TODO(ROX-29454) figure out how to get query like `select count(distinct (name, version, operatingsystem)) from image_component_v2;`
	results, err := v.Get(queryCtx, clonedQ)
	if err != nil {
		return 0, err
	}

	return len(results), nil
}

func (v *imageComponentFlatViewImpl) Get(ctx context.Context, q *v1.Query) ([]ComponentFlat, error) {
	if err := common.ValidateQuery(q); err != nil {
		return nil, err
	}

	var err error
	// Avoid changing the passed query
	cloned := q.CloneVT()
	// Update the sort options to use aggregations if necessary as we are grouping by CVEs
	cloned = common.UpdateSortAggs(cloned)
	cloned, err = common.WithSACFilter(ctx, resources.Image, cloned)
	if err != nil {
		return nil, err
	}

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	ret := make([]ComponentFlat, 0, paginated.GetLimit(q.GetPagination().GetLimit(), 100))
	err = pgSearch.RunSelectRequestForSchemaFn[imageComponentFlatResponse](queryCtx, v.db, v.schema, withSelectComponentCoreResponseQuery(cloned), func(r *imageComponentFlatResponse) error {
		// For each record, sort the IDs so that result looks consistent.
		sort.SliceStable(r.ComponentIDs, func(i, j int) bool {
			return r.ComponentIDs[i] < r.ComponentIDs[j]
		})
		ret = append(ret, r)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func withSelectComponentCoreResponseQuery(q *v1.Query) *v1.Query {
	cloned := q.CloneVT()

	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.Component).Proto(),
		search.NewQuerySelect(search.ComponentID).Distinct().Proto(),
		search.NewQuerySelect(search.ComponentVersion).Proto(),
		search.NewQuerySelect(search.OperatingSystem).Proto(),
		search.NewQuerySelect(search.ComponentTopCVSS).AggrFunc(aggregatefunc.Max).Proto(),
		search.NewQuerySelect(search.ComponentRiskScore).AggrFunc(aggregatefunc.Max).Proto(),
	}

	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{search.Component.String(), search.ComponentVersion.String(), search.OperatingSystem.String()},
	}

	return cloned
}
