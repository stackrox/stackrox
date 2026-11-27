package imagecveflat

import (
	"context"
	"sort"

	imagev2common "github.com/stackrox/rox/central/imagev2/common"
	"github.com/stackrox/rox/central/views"
	"github.com/stackrox/rox/central/views/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
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
	q = imagev2common.WithRowsFromImageV2Only(q)

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	return pgSearch.RunDistinctCountForSchema(queryCtx, v.db, v.schema, q, search.CVE)
}

func (v *imageCVEFlatViewImpl) Get(ctx context.Context, q *v1.Query, options views.ReadOptions) ([]CveFlat, error) {
	if err := common.ValidateQuery(q); err != nil {
		return nil, err
	}
	q = imagev2common.WithRowsFromImageV2Only(q)

	// Avoid changing the passed query
	cloned := q.CloneVT()
	// A severity sort resolves to max(severity), which is low-cardinality; add tiebreakers so
	// pagination is deterministic before converting sorts to aggregations.
	cloned = common.WithMaxSeveritySortTiebreakers(cloned)
	// Update the sort options to use aggregations if necessary as we are grouping by CVEs
	cloned = common.UpdateSortAggs(cloned)

	// Performance improvements to narrow aggregations performed
	var cvesToFilter []string
	var err error
	if cloned.GetPagination().GetLimit() > 0 || cloned.GetPagination().GetOffset() > 0 {
		cvesToFilter, err = v.getFilteredCVEs(ctx, cloned)
		if err != nil {
			log.Error(err)
			return nil, err
		}

		if cloned.GetPagination() != nil && cloned.GetPagination().GetSortOptions() != nil {
			// The CVE list that we get from the above query is paginated. So when we fetch the details and aggregates for those CVEs,
			// we do not need to re-apply pagination limit and offset
			cloned.Pagination = &v1.QueryPagination{SortOptions: cloned.GetPagination().GetSortOptions()}
		}
	}

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	var ret []CveFlat
	err = pgSearch.RunSelectRequestForSchemaFn[imageCVEFlatResponse](queryCtx, v.db, v.schema, withSelectCVEFlatResponseQuery(cloned, cvesToFilter, options), func(r *imageCVEFlatResponse) error {
		// For each record, sort the IDs so that result looks consistent.
		sort.SliceStable(r.CVEIDs, func(i, j int) bool {
			return r.CVEIDs[i] < r.CVEIDs[j]
		})
		ret = append(ret, r)
		return nil
	})
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return ret, nil
}

// withSelectCVEStringsQuery builds the lightweight phase-1 paging query that
// selects only the CVE string (the generator adds the max() aggregates needed
// by the sort), grouped by CVE. It deliberately does NOT aggregate the per-CVE
// instance IDs via jsonb_agg(DISTINCT id): a DISTINCT aggregate forces a sorted
// GroupAggregate over the entire image_cves_v2 fan-out, which spills to disk at
// scale (observed ~93s / 649MB at 100k images / 5M CVE rows). max()-only
// aggregation uses an in-memory HashAggregate with no such sort. The selected
// CVE strings then scope the phase-2 detail query (withSelectCVEFlatResponseQuery).
func withSelectCVEStringsQuery(q *v1.Query) *v1.Query {
	cloned := q.CloneVT()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.CVE).Proto(),
	}
	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{search.CVE.String()},
	}

	return cloned
}

func withSelectCVEFlatResponseQuery(q *v1.Query, cvesToFilter []string, options views.ReadOptions) *v1.Query {
	cloned := q.CloneVT()
	if len(cvesToFilter) > 0 {
		// Scope the detail/aggregate query to the CVEs selected by phase 1. The
		// full joins and WHERE clause are re-applied here, so the aggregated
		// values match exactly what filtering by instance IDs would have produced.
		cloned = search.ConjunctionQuery(cloned, search.NewQueryBuilder().AddExactMatches(search.CVE, cvesToFilter...).ProtoQuery())
		cloned.Pagination = q.GetPagination()
	}

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
	var searchField search.FieldLabel
	if features.FlattenImageData.Enabled() {
		searchField = search.ImageID
	} else {
		searchField = search.ImageSHA
	}
	if !options.SkipGetAffectedImages {
		cloned.Selects = append(cloned.Selects, search.NewQuerySelect(searchField).AggrFunc(aggregatefunc.Count).Distinct().Proto())
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

	return cloned
}

// getFilteredCVEs runs phase 1: it returns the page of CVE strings (ordered by
// the requested sort, with limit/offset applied) that the phase-2 detail query
// is then scoped to. See withSelectCVEStringsQuery for why it returns CVE
// strings rather than instance IDs.
func (v *imageCVEFlatViewImpl) getFilteredCVEs(ctx context.Context, q *v1.Query) ([]string, error) {
	var cvesToFilter []string

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	err := pgSearch.RunSelectRequestForSchemaFn[imageCVEFlatResponse](queryCtx, v.db, v.schema, withSelectCVEStringsQuery(q), func(r *imageCVEFlatResponse) error {
		cvesToFilter = append(cvesToFilter, r.CVE)
		return nil
	})
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return cvesToFilter, nil
}
