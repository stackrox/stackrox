package imagecomponentflat

import (
	"context"
	"sort"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/views/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	"github.com/stackrox/rox/pkg/utils"
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

	var results []*imageComponentFlatCount
	results, err = pgSearch.RunSelectRequestForSchema[imageComponentFlatCount](queryCtx, v.db, v.schema, common.WithCountQuery(q, search.CVE))
	if err != nil {
		return 0, err
	}
	if len(results) == 0 {
		return 0, nil
	}
	if len(results) > 1 {
		err = errors.Errorf("Retrieved multiple rows when only one row is expected for count query %q", q.String())
		utils.Should(err)
		return 0, err
	}
	return results[0].ComponentCount, nil
}

func (v *imageComponentFlatViewImpl) Get(ctx context.Context, q *v1.Query) ([]ComponentFlat, error) {
	if err := common.ValidateQuery(q); err != nil {
		return nil, err
	}

	var err error
	// Avoid changing the passed query
	cloned := q.CloneVT()
	cloned, err = common.WithSACFilter(ctx, resources.Image, cloned)
	if err != nil {
		return nil, err
	}

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	var results []*imageComponentFlatResponse
	results, err = pgSearch.RunSelectRequestForSchema[imageComponentFlatResponse](queryCtx, v.db, v.schema, withSelectComponentCoreResponseQuery(cloned))
	if err != nil {
		return nil, err
	}

	ret := make([]ComponentFlat, 0, len(results))
	for _, r := range results {
		// For each record, sort the IDs so that result looks consistent.
		sort.SliceStable(r.ComponentIDs, func(i, j int) bool {
			return r.ComponentIDs[i] < r.ComponentIDs[j]
		})
		ret = append(ret, r)
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
		search.NewQuerySelect(search.ComponentPriority).AggrFunc(aggregatefunc.Min).Proto(),
	}

	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{search.Component.String(), search.ComponentVersion.String(), search.OperatingSystem.String()},
	}

	// This is to minimize UI change and hide an implementation detail that the schema is denormalized.
	// Now that these fields are aggregations, in order to sort on them, we have to set the sort field as such to match
	// the query field.
	for _, sortOption := range cloned.GetPagination().GetSortOptions() {
		if sortOption.Field == search.ComponentTopCVSS.String() {
			sortOption.Field = search.ComponentTopCVSSMax.String()
		}
		if sortOption.Field == search.ComponentPriority.String() {
			sortOption.Field = search.ComponentPriorityMin.String()
		}
	}

	return cloned
}
