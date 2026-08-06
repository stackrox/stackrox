package imagecve

import (
	"context"
	"sort"

	"github.com/pkg/errors"
	imagev2common "github.com/stackrox/rox/central/imagev2/common"
	"github.com/stackrox/rox/central/views"
	"github.com/stackrox/rox/central/views/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
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
	q = imagev2common.WithRowsFromImageV2Only(q)

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	return pgSearch.RunDistinctCountForSchema(queryCtx, v.db, v.schema, q, search.CVE)
}

func (v *imageCVECoreViewImpl) CountBySeverity(ctx context.Context, q *v1.Query) (common.ResourceCountByCVESeverity, error) {
	if err := common.ValidateQuery(q); err != nil {
		return nil, err
	}
	q = imagev2common.WithRowsFromImageV2Only(q)

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
	q = imagev2common.WithRowsFromImageV2Only(q)

	// Avoid changing the passed query
	cloned := q.CloneVT()
	// Update the sort options to use aggregations if necessary as we are grouping by CVEs
	cloned = common.UpdateSortAggs(cloned)

	var cvesToFilter []string
	var err error
	if cloned.GetPagination().GetLimit() > 0 || cloned.GetPagination().GetOffset() > 0 {
		cvesToFilter, err = v.getFilteredCVEs(ctx, cloned)
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
	err = pgSearch.RunSelectRequestForSchemaFn[imageCVECoreResponse](queryCtx, v.db, v.schema, withSelectCVECoreResponseQuery(cloned, cvesToFilter, options), func(r *imageCVECoreResponse) error {
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

func (v *imageCVECoreViewImpl) TopSeverityBatch(ctx context.Context, entityIDs []string, entityType search.FieldLabel, q *v1.Query) (map[string]storage.VulnerabilitySeverity, error) {
	if len(entityIDs) == 0 {
		return nil, nil
	}
	filtered := imagev2common.WithRowsFromImageV2Only(q.CloneVT())
	cloned := search.ConjunctionQuery(filtered,
		search.NewQueryBuilder().AddExactMatches(entityType, entityIDs...).ProtoQuery(),
	)
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(entityType).Proto(),
		search.NewQuerySelect(search.Severity).AggrFunc(aggregatefunc.Max).Proto(),
	}
	cloned.GroupBy = &v1.QueryGroupBy{Fields: []string{entityType.String()}}
	cloned.Pagination = nil

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	result := make(map[string]storage.VulnerabilitySeverity, len(entityIDs))
	var err error
	switch entityType {
	case search.ImageID:
		err = pgSearch.RunSelectRequestForSchemaFn[imageSeverityResult](queryCtx, v.db, v.schema, cloned, func(r *imageSeverityResult) error {
			result[r.EntityID] = r.TopSeverity
			return nil
		})
	case search.DeploymentID:
		err = pgSearch.RunSelectRequestForSchemaFn[deploymentSeverityResult](queryCtx, v.db, v.schema, cloned, func(r *deploymentSeverityResult) error {
			result[r.EntityID] = r.TopSeverity
			return nil
		})
	default:
		return nil, errors.Errorf("unsupported entity type for TopSeverityBatch: %s", entityType)
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (v *imageCVECoreViewImpl) GetDeploymentIDs(ctx context.Context, q *v1.Query) ([]string, error) {
	q.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.DeploymentID).Distinct().Proto(),
	}

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	ret := make([]string, 0, paginated.GetLimit(q.GetPagination().GetLimit(), 100))
	err := pgSearch.RunSelectRequestForSchemaFn[deploymentResponse](queryCtx, v.db, v.schema, q, func(r *deploymentResponse) error {
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
	var err error
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
		search.NewQuerySelect(search.CVE).Proto(),
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
			common.WithCountBySeverityAndFixabilityQuery(q, searchField).GetSelects()...,
		)
	}

	return cloned
}

func withSelectCVECoreResponseQuery(q *v1.Query, cvesToFilter []string, options views.ReadOptions) *v1.Query {
	cloned := q.CloneVT()
	if len(cvesToFilter) > 0 {
		cloned = search.ConjunctionQuery(cloned, search.NewQueryBuilder().AddExactMatches(search.CVE, cvesToFilter...).ProtoQuery())
		cloned.Pagination = q.GetPagination()
	}
	searchField := search.ImageSHA
	if features.FlattenImageData.Enabled() {
		searchField = search.ImageID
	}
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.CVE).Proto(),
		search.NewQuerySelect(search.CVEID).Distinct().Proto(),
		search.NewQuerySelect(search.Severity).AggrFunc(aggregatefunc.Max).Proto(),
		search.NewQuerySelect(search.EPSSProbablity).AggrFunc(aggregatefunc.Max).Proto(),
	}
	if !options.SkipGetImagesBySeverity {
		cloned.Selects = append(cloned.Selects,
			common.WithCountBySeverityAndFixabilityQuery(q, searchField).GetSelects()...,
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

type cveNameResponse struct {
	CVE string `db:"cve"`
}

func (v *imageCVECoreViewImpl) getFilteredCVEs(ctx context.Context, q *v1.Query) ([]string, error) {
	var cvesToFilter []string

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	err := pgSearch.RunSelectRequestForSchemaFn[cveNameResponse](queryCtx, v.db, v.schema, withSelectCVEIdentifiersQuery(q), func(r *cveNameResponse) error {
		cvesToFilter = append(cvesToFilter, r.CVE)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return cvesToFilter, nil
}
