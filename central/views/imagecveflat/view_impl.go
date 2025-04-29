package imagecveflat

import (
	"context"
	"sort"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/views"
	"github.com/stackrox/rox/central/views/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
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
	log          = logging.LoggerForModule()
)

type imageCVEFlatViewImpl struct {
	schema *walker.Schema
	db     postgres.DB
}

func (v *imageCVEFlatViewImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
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

	var results []*imageCVEFlatCount
	results, err = pgSearch.RunSelectRequestForSchema[imageCVEFlatCount](queryCtx, v.db, v.schema, common.WithCountQuery(q, search.CVE))
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
	return results[0].CVECount, nil
}

func (v *imageCVEFlatViewImpl) Get(ctx context.Context, q *v1.Query, options views.ReadOptions) ([]CveFlat, error) {
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

	log.Infof("SHREWS -- IN flat")
	var results []*imageCVEFlatResponse
	results, err = pgSearch.RunSelectRequestForSchema[imageCVEFlatResponse](queryCtx, v.db, v.schema, withSelectCVECoreResponseQuery(cloned, options))
	if err != nil {
		log.Infof("SHREWS -- IN flat %v", err)
		return nil, err
	}

	log.Infof("SHREWS -- IN flat after query")
	ret := make([]CveFlat, 0, len(results))
	for _, r := range results {
		// For each record, sort the IDs so that result looks consistent.
		sort.SliceStable(r.CVEIDs, func(i, j int) bool {
			return r.CVEIDs[i] < r.CVEIDs[j]
		})
		ret = append(ret, r)
	}
	return ret, nil
}

func withSelectCVECoreResponseQuery(q *v1.Query, options views.ReadOptions) *v1.Query {
	cloned := q.CloneVT()

	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.CVE).Proto(),
		search.NewQuerySelect(search.CVEID).Distinct().Proto(),
		search.NewQuerySelect(search.EPSSProbablity).AggrFunc(aggregatefunc.Max).Proto(),
		search.NewQuerySelect(search.ImpactScore).AggrFunc(aggregatefunc.Max).Proto(),
		search.NewQuerySelect(search.FirstImageOccurrenceTimestamp).AggrFunc(aggregatefunc.Min).Proto(),
		search.NewQuerySelect(search.VulnerabilityState).AggrFunc(aggregatefunc.Max).Proto(),
		search.NewQuerySelect(search.Severity).AggrFunc(aggregatefunc.Max).Proto(),
	}
	if !options.SkipGetTopCVSS {
		cloned.Selects = append(cloned.Selects, search.NewQuerySelect(search.CVSS).AggrFunc(aggregatefunc.Max).Proto())
	}
	if !options.SkipGetAffectedImages {
		cloned.Selects = append(cloned.Selects, search.NewQuerySelect(search.ImageSHA).AggrFunc(aggregatefunc.Count).Distinct().Proto())
	}
	if !options.SkipGetFirstDiscoveredInSystem {
		cloned.Selects = append(cloned.Selects, search.NewQuerySelect(search.CVECreatedTime).AggrFunc(aggregatefunc.Min).Proto())
	}
	if !options.SkipPublishedDate {
		cloned.Selects = append(cloned.Selects, search.NewQuerySelect(search.CVEPublishedOn).AggrFunc(aggregatefunc.Min).Proto())
	}
	if !options.SkipGetTopNVDCVSS {
		cloned.Selects = append(cloned.Selects, search.NewQuerySelect(search.NVDCVSS).AggrFunc(aggregatefunc.Max).Proto())
	}

	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{search.CVE.String()},
	}

	// This is to minimize UI change and hide an implementation detail that the schema is denormalized.
	// Now that these fields are aggregations, in order to sort on them, we have to set the sort field as such to match
	// the query field.
	for _, sortOption := range cloned.GetPagination().GetSortOptions() {
		if sortOption.Field == search.Severity.String() {
			sortOption.Field = search.SeverityMax.String()
		}
		if sortOption.Field == search.CVSS.String() {
			sortOption.Field = search.CVSSMax.String()
		}
	}

	return cloned
}
