package imagecve

import (
	"context"
	"sort"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/views"
	"github.com/stackrox/rox/central/views/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	"github.com/stackrox/rox/pkg/utils"
)

type imageCVECoreViewImpl struct {
	schema *walker.Schema
	db     postgres.DB
}

func (v *imageCVECoreViewImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	if err := validateQuery(q); err != nil {
		return 0, err
	}

	var err error
	q, err = withSACFilter(ctx, resources.Image, q)
	if err != nil {
		return 0, err
	}

	var results []*imageCVECoreCount
	results, err = pgSearch.RunSelectRequestForSchema[imageCVECoreCount](ctx, v.db, v.schema, withCountQuery(q))
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
	if err := validateQuery(q); err != nil {
		return nil, err
	}

	var err error
	q, err = withSACFilter(ctx, resources.Image, q)
	if err != nil {
		return nil, err
	}

	var results []*resourceCountByImageCVESeverity
	results, err = pgSearch.RunSelectRequestForSchema[resourceCountByImageCVESeverity](ctx, v.db, v.schema, withCountBySeveritySelectQuery(q, search.CVE))
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return &resourceCountByImageCVESeverity{}, nil
	}
	if len(results) > 1 {
		err = errors.Errorf("Retrieved multiple rows when only one row is expected for count query %q", q.String())
		utils.Should(err)
		return &resourceCountByImageCVESeverity{}, err
	}

	return &resourceCountByImageCVESeverity{
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
	if err := validateQuery(q); err != nil {
		return nil, err
	}

	var err error
	q, err = withSACFilter(ctx, resources.Image, q)
	if err != nil {
		return nil, err
	}

	var results []*imageCVECoreResponse
	results, err = pgSearch.RunSelectRequestForSchema[imageCVECoreResponse](ctx, v.db, v.schema, withSelectQuery(q, options))
	if err != nil {
		return nil, err
	}

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
	q, err = withSACFilter(ctx, resources.Deployment, q)
	if err != nil {
		return nil, err
	}

	q.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.DeploymentID).Distinct().Proto(),
	}

	var results []*deploymentResponse
	results, err = pgSearch.RunSelectRequestForSchema[deploymentResponse](ctx, v.db, v.schema, q)
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
	q, err = withSACFilter(ctx, resources.Image, q)
	if err != nil {
		return nil, err
	}

	q.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.ImageSHA).Distinct().Proto(),
	}

	var results []*imageResponse
	results, err = pgSearch.RunSelectRequestForSchema[imageResponse](ctx, v.db, v.schema, q)
	if err != nil || len(results) == 0 {
		return nil, err
	}

	ret := make([]string, 0, len(results))
	for _, r := range results {
		ret = append(ret, r.ImageID)
	}
	return ret, nil
}

func validateQuery(q *v1.Query) error {
	// We only support a dynamic where clause. CveCore has a pre-defined select and group by. Remember this is a "view".
	if len(q.GetSelects()) > 0 {
		return errors.Errorf("Unexpected select clause in query %q", q.String())
	}
	if q.GetGroupBy() != nil {
		return errors.Errorf("Unexpected group by clause in query %q", q.String())
	}
	return nil
}

func withSelectQuery(q *v1.Query, options views.ReadOptions) *v1.Query {
	cloned := q.Clone()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.CVE).Proto(),
		search.NewQuerySelect(search.CVEID).Distinct().Proto(),
	}
	if !options.SkipGetImagesBySeverity {
		cloned.Selects = append(cloned.Selects,
			withCountBySeveritySelectQuery(q, search.ImageSHA).Selects...,
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
	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{search.CVE.String()},
	}
	return cloned
}

func withCountBySeveritySelectQuery(q *v1.Query, countOn search.FieldLabel) *v1.Query {
	cloned := q.Clone()
	cloned.Selects = append(cloned.Selects,
		search.NewQuerySelect(countOn).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter("critical_severity_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.Severity,
						storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter("fixable_critical_severity_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.Severity,
						storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY.String(),
					).
					AddBools(search.Fixable, true).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter("important_severity_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.Severity,
						storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter("fixable_important_severity_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.Severity,
						storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY.String(),
					).
					AddBools(search.Fixable, true).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter("moderate_severity_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.Severity,
						storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter("fixable_moderate_severity_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.Severity,
						storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY.String(),
					).
					AddBools(search.Fixable, true).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter("low_severity_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.Severity,
						storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			Distinct().
			AggrFunc(aggregatefunc.Count).
			Filter("fixable_low_severity_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.Severity,
						storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY.String(),
					).
					AddBools(search.Fixable, true).ProtoQuery(),
			).Proto(),
	)
	return cloned
}

func withCountQuery(q *v1.Query) *v1.Query {
	cloned := q.Clone()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.CVE).AggrFunc(aggregatefunc.Count).Distinct().Proto(),
	}
	return cloned
}

func withSACFilter(ctx context.Context, targetResource permissions.ResourceMetadata, query *v1.Query) (*v1.Query, error) {
	var sacQueryFilter *v1.Query

	scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_ACCESS).Resource(targetResource)
	scopeTree, err := scopeChecker.EffectiveAccessScope(permissions.View(targetResource))
	if err != nil {
		return nil, err
	}
	sacQueryFilter, err = sac.BuildNonVerboseClusterNamespaceLevelSACQueryFilter(scopeTree)
	if err != nil {
		return nil, err
	}

	return search.FilterQueryByQuery(query, sacQueryFilter), nil
}
