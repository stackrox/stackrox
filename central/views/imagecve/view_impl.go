package imagecve

import (
	"context"
	"sort"

	"github.com/stackrox/rox/central/views"
	"github.com/stackrox/rox/central/views/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
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

type imageCVECoreViewImpl struct {
	schema *walker.Schema
	db     postgres.DB
}

func (v *imageCVECoreViewImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
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

	result, err := pgSearch.RunSelectOneForSchema[imageCVECoreCount](queryCtx, v.db, v.schema, common.WithCountQuery(q, search.CVE))
	if err != nil {
		return 0, err
	}
	if result == nil {
		return 0, nil
	}
	return result.CVECount, nil
}

func (v *imageCVECoreViewImpl) CountBySeverity(ctx context.Context, q *v1.Query) (common.ResourceCountByCVESeverity, error) {
	if err := common.ValidateQuery(q); err != nil {
		return nil, err
	}

	var err error
	q, err = common.WithSACFilter(ctx, resources.Image, q)
	if err != nil {
		return nil, err
	}

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	result, err := pgSearch.RunSelectOneForSchema[common.ResourceCountByImageCVESeverity](queryCtx, v.db, v.schema, common.WithCountBySeverityAndFixabilityQuery(q, search.CVE))
	if err != nil {
		return nil, err
	}
	if result == nil {
		return &common.ResourceCountByImageCVESeverity{}, nil
	}

	return &common.ResourceCountByImageCVESeverity{
		CriticalSeverityCount:        result.CriticalSeverityCount,
		FixableCriticalSeverityCount: result.FixableCriticalSeverityCount,

		ImportantSeverityCount:        result.ImportantSeverityCount,
		FixableImportantSeverityCount: result.FixableImportantSeverityCount,

		ModerateSeverityCount:        result.ModerateSeverityCount,
		FixableModerateSeverityCount: result.FixableModerateSeverityCount,

		LowSeverityCount:        result.LowSeverityCount,
		FixableLowSeverityCount: result.FixableLowSeverityCount,

		UnknownSeverityCount:        result.UnknownSeverityCount,
		FixableUnknownSeverityCount: result.FixableUnknownSeverityCount,
	}, nil
}

func (v *imageCVECoreViewImpl) Get(ctx context.Context, q *v1.Query, options views.ReadOptions) ([]CveCore, error) {
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

	var cveIDsToFilter []string
	if cloned.GetPagination().GetLimit() > 0 || cloned.GetPagination().GetOffset() > 0 {
		cveIDsToFilter, err = v.getFilteredCVEs(ctx, cloned)
		if err != nil {
			return nil, err
		}

		if cloned.GetPagination() != nil && cloned.GetPagination().GetSortOptions() != nil {
			// The CVE ID list that we get from the above query is paginated. So when we fetch the details and aggregates for those CVEs,
			// we do not need to re-apply pagination limit and offset
			cloned.Pagination = &v1.QueryPagination{SortOptions: cloned.GetPagination().GetSortOptions()}
		}
	}
	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	ret := make([]CveCore, 0, paginated.GetLimit(q.GetPagination().GetLimit(), 100))
	err = pgSearch.RunSelectRequestForSchemaFn[imageCVECoreResponse](queryCtx, v.db, v.schema, withSelectCVECoreResponseQuery(cloned, cveIDsToFilter, options), func(r *imageCVECoreResponse) error {
		// For each record, sort the IDs so that result looks consistent.
		sort.SliceStable(r.CVEIDs, func(i, j int) bool {
			return r.CVEIDs[i] < r.CVEIDs[j]
		})
		ret = append(ret, r)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (v *imageCVECoreViewImpl) GetDeploymentIDs(ctx context.Context, q *v1.Query) ([]string, error) {
	var err error
	q, err = common.WithSACFilter(ctx, resources.Deployment, q)
	if err != nil {
		return nil, err
	}

	q.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.DeploymentID).Distinct().Proto(),
	}

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	ret := make([]string, 0, paginated.GetLimit(q.GetPagination().GetLimit(), 100))
	err = pgSearch.RunSelectRequestForSchemaFn[deploymentResponse](queryCtx, v.db, v.schema, q, func(r *deploymentResponse) error {
		ret = append(ret, r.DeploymentID)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(ret) == 0 {
		return nil, nil
	}
	return ret, nil
}

func (v *imageCVECoreViewImpl) GetImageIDs(ctx context.Context, q *v1.Query) ([]string, error) {
	var err error
	q, err = common.WithSACFilter(ctx, resources.Image, q)
	if err != nil {
		return nil, err
	}

	searchField := search.ImageSHA
	if features.FlattenImageData.Enabled() {
		searchField = search.ImageID
	}
	q.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(searchField).Distinct().Proto(),
	}

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	ret := make([]string, 0, paginated.GetLimit(q.GetPagination().GetLimit(), 100))
	if features.FlattenImageData.Enabled() {
		err = pgSearch.RunSelectRequestForSchemaFn[imageV2Response](queryCtx, v.db, v.schema, q, func(r *imageV2Response) error {
			ret = append(ret, r.ImageID)
			return nil
		})
	} else {
		err = pgSearch.RunSelectRequestForSchemaFn[imageResponse](queryCtx, v.db, v.schema, q, func(r *imageResponse) error {
			ret = append(ret, r.ImageID)
			return nil
		})
	}
	if err != nil {
		return nil, err
	}
	if len(ret) == 0 {
		return nil, nil
	}
	return ret, nil
}

func withSelectCVEIdentifiersQuery(q *v1.Query) *v1.Query {
	searchField := search.ImageSHA
	if features.FlattenImageData.Enabled() {
		searchField = search.ImageID
	}
	cloned := q.CloneVT()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.CVEID).Distinct().Proto(),
	}
	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{search.CVE.String()},
	}

	// For pagination and sort to work properly, the filter query to get the CVEs needs to
	// include the fields we are sorting on.  At this time custom code is required when
	// sorting on custom sort fields.  For instance counts on the Severity column based on
	// a value of that column
	// TODO(ROX-26310): Update the search framework to inject required select.
	// Add the severity selects if severity is a sort option to ensure we have the filtered
	// list of CVEs ordered appropriately.
	if common.IsSortBySeverityCounts(cloned) {
		cloned.Selects = append(cloned.Selects,
			common.WithCountBySeverityAndFixabilityQuery(q, searchField).Selects...,
		)
	}

	return cloned
}

func withSelectCVECoreResponseQuery(q *v1.Query, cveIDsToFilter []string, options views.ReadOptions) *v1.Query {
	cloned := q.CloneVT()
	if len(cveIDsToFilter) > 0 {
		cloned = search.ConjunctionQuery(cloned, search.NewQueryBuilder().AddDocIDs(cveIDsToFilter...).ProtoQuery())
		cloned.Pagination = q.GetPagination()
	}
	searchField := search.ImageSHA
	if features.FlattenImageData.Enabled() {
		searchField = search.ImageID
	}
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.CVE).Proto(),
		search.NewQuerySelect(search.CVEID).Distinct().Proto(),
	}
	if !options.SkipGetImagesBySeverity {
		cloned.Selects = append(cloned.Selects,
			common.WithCountBySeverityAndFixabilityQuery(q, searchField).Selects...,
		)
	}
	if !options.SkipGetTopCVSS {
		cloned.Selects = append(cloned.Selects, search.NewQuerySelect(search.CVSS).AggrFunc(aggregatefunc.Max).Proto())
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

func (v *imageCVECoreViewImpl) getFilteredCVEs(ctx context.Context, q *v1.Query) ([]string, error) {
	var cveIDsToFilter []string

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	// TODO(@charmik) : Update the SQL query generator to not include 'ORDER BY' and 'GROUP BY' fields in the select clause (before where).
	//  SQL syntax does not need those fields in the select clause. The below query for example would work fine
	//  "SELECT JSONB_AGG(DISTINCT(image_cves.Id)) AS cve_id FROM image_cves GROUP BY image_cves.CveBaseInfo_Cve ORDER BY MAX(image_cves.Cvss) DESC LIMIT 20;"
	err := pgSearch.RunSelectRequestForSchemaFn[imageCVECoreResponse](queryCtx, v.db, v.schema, withSelectCVEIdentifiersQuery(q), func(r *imageCVECoreResponse) error {
		cveIDsToFilter = append(cveIDsToFilter, r.CVEIDs...)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return cveIDsToFilter, nil
}
