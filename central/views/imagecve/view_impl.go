package imagecve

import (
	"context"
	"sort"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/views"
	"github.com/stackrox/rox/central/views/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
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

	log = logging.LoggerForModule()
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

	var results []*imageCVECoreCount
	results, err = pgSearch.RunSelectRequestForSchema[imageCVECoreCount](queryCtx, v.db, v.schema, common.WithCountQuery(q, search.CVE))
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

	var results []*common.ResourceCountByImageCVESeverity
	results, err = pgSearch.RunSelectRequestForSchema[common.ResourceCountByImageCVESeverity](queryCtx, v.db, v.schema, common.WithCountBySeverityAndFixabilityQuery(q, search.CVE))
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return &common.ResourceCountByImageCVESeverity{}, nil
	}
	if len(results) > 1 {
		err = errors.Errorf("Retrieved multiple rows when only one row is expected for count query %q", q.String())
		utils.Should(err)
		return &common.ResourceCountByImageCVESeverity{}, err
	}

	return &common.ResourceCountByImageCVESeverity{
		CriticalSeverityCount:        results[0].CriticalSeverityCount,
		FixableCriticalSeverityCount: results[0].FixableCriticalSeverityCount,

		ImportantSeverityCount:        results[0].ImportantSeverityCount,
		FixableImportantSeverityCount: results[0].FixableImportantSeverityCount,

		ModerateSeverityCount:        results[0].ModerateSeverityCount,
		FixableModerateSeverityCount: results[0].FixableModerateSeverityCount,

		LowSeverityCount:        results[0].LowSeverityCount,
		FixableLowSeverityCount: results[0].FixableLowSeverityCount,
	}, nil
}

func (v *imageCVECoreViewImpl) Get(ctx context.Context, q *v1.Query, options views.ReadOptions) ([]CveCore, error) {
	log.Info("SHREWS -- Get")
	if err := common.ValidateQuery(q); err != nil {
		return nil, err
	}

	log.Info("SHREWS -- Get -- query is valid")
	var err error
	// Avoid changing the passed query
	cloned := q.CloneVT()
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
		log.Infof("SHREWS -- Get -- filter ID length %d", len(cveIDsToFilter))
		log.Infof("SHREWS -- Get -- filter ID length %v", cveIDsToFilter)

		if cloned.GetPagination() != nil && cloned.GetPagination().GetSortOptions() != nil {
			// The CVE ID list that we get from the above query is paginated. So when we fetch the details and aggregates for those CVEs,
			// we do not need to re-apply pagination limit and offset
			cloned.Pagination = &v1.QueryPagination{SortOptions: cloned.GetPagination().GetSortOptions()}
		}
	}
	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	var results []*imageCVECoreResponse
	results, err = pgSearch.RunSelectRequestForSchema[imageCVECoreResponse](queryCtx, v.db, v.schema, withSelectCVECoreResponseQuery(cloned, cveIDsToFilter, options))
	if err != nil {
		return nil, err
	}
	log.Infof("SHREWS -- Get -- results length %d", len(results))

	ret := make([]CveCore, 0, len(results))
	for _, r := range results {
		// For each record, sort the IDs so that result looks consistent.
		sort.SliceStable(r.CVEIDs, func(i, j int) bool {
			return r.CVEIDs[i] < r.CVEIDs[j]
		})
		ret = append(ret, r)
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

	var results []*deploymentResponse
	results, err = pgSearch.RunSelectRequestForSchema[deploymentResponse](queryCtx, v.db, v.schema, q)
	if err != nil || len(results) == 0 {
		return nil, err
	}

	ret := make([]string, 0, len(results))
	for _, r := range results {
		ret = append(ret, r.DeploymentID)
	}
	return ret, nil
}

func (v *imageCVECoreViewImpl) GetImageIDs(ctx context.Context, q *v1.Query) ([]string, error) {
	var err error
	q, err = common.WithSACFilter(ctx, resources.Image, q)
	if err != nil {
		return nil, err
	}

	q.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.ImageSHA).Distinct().Proto(),
	}

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	var results []*imageResponse
	results, err = pgSearch.RunSelectRequestForSchema[imageResponse](queryCtx, v.db, v.schema, q)
	if err != nil || len(results) == 0 {
		return nil, err
	}

	ret := make([]string, 0, len(results))
	for _, r := range results {
		ret = append(ret, r.ImageID)
	}
	return ret, nil
}

func withSelectCVEIdentifiersQuery(q *v1.Query) *v1.Query {
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
			common.WithCountBySeverityAndFixabilityQuery(q, search.ImageSHA).Selects...,
		)
	}

	return cloned
}

func withSelectCVECoreResponseQuery(q *v1.Query, cveIDsToFilter []string, options views.ReadOptions) *v1.Query {
	cloned := q.CloneVT()
	if len(cveIDsToFilter) > 0 {
		if features.FlattenCVEData.Enabled() {
			cloned = search.ConjunctionQuery(cloned, search.NewQueryBuilder().AddExactMatches(search.CVEID, cveIDsToFilter...).ProtoQuery())
		} else {
			cloned = search.ConjunctionQuery(cloned, search.NewQueryBuilder().AddDocIDs(cveIDsToFilter...).ProtoQuery())
		}
		cloned.Pagination = q.GetPagination()
	}
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.CVE).Proto(),
		search.NewQuerySelect(search.CVEID).Distinct().Proto(),
	}
	if !options.SkipGetImagesBySeverity {
		cloned.Selects = append(cloned.Selects,
			common.WithCountBySeverityAndFixabilityQuery(q, search.ImageSHA).Selects...,
		)
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
	return cloned
}

func (v *imageCVECoreViewImpl) getFilteredCVEs(ctx context.Context, q *v1.Query) ([]string, error) {
	var cveIDsToFilter []string

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	// TODO(@charmik) : Update the SQL query generator to not include 'ORDER BY' and 'GROUP BY' fields in the select clause (before where).
	//  SQL syntax does not need those fields in the select clause. The below query for example would work fine
	//  "SELECT JSONB_AGG(DISTINCT(image_cves.Id)) AS cve_id FROM image_cves GROUP BY image_cves.CveBaseInfo_Cve ORDER BY MAX(image_cves.Cvss) DESC LIMIT 20;"
	var identifiersList []*imageCVECoreResponse
	identifiersList, err := pgSearch.RunSelectRequestForSchema[imageCVECoreResponse](queryCtx, v.db, v.schema, withSelectCVEIdentifiersQuery(q))
	if err != nil {
		return nil, err
	}

	for _, idList := range identifiersList {
		cveIDsToFilter = append(cveIDsToFilter, idList.CVEIDs...)
	}

	return cveIDsToFilter, nil
}
