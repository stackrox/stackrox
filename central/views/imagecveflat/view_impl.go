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

const (
	knownExploitCountAlias    = "known_exploit_count"
	knownRansomwareCountAlias = "known_ransomware_campaign_count"
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
	// Update the sort options to use aggregations if necessary as we are grouping by CVEs
	cloned = common.UpdateSortAggs(cloned)

	// Performance improvements to narrow aggregations performed
	var cvesToFilter []string
	var err error
	if cloned.GetPagination().GetLimit() > 0 || cloned.GetPagination().GetOffset() > 0 {
		cvesToFilter, err = v.getFilteredCVEs(ctx, cloned, options.UseCVENameFiltering)
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

func withSelectCVEIdentifiersQuery(q *v1.Query) *v1.Query {
	cloned := q.CloneVT()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.CVEID).Distinct().Proto(),
	}
	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{search.CVE.String()},
	}

	return cloned
}

// withSelectCVENamesQuery selects the CVE name, which is already the group key, instead
// of aggregating row IDs with jsonb_agg. This is substantially cheaper on large tables.
func withSelectCVENamesQuery(q *v1.Query) *v1.Query {
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
		var filter *v1.Query
		if options.UseCVENameFiltering {
			filter = search.NewQueryBuilder().AddExactMatches(search.CVE, cvesToFilter...).ProtoQuery()
		} else {
			filter = search.NewQueryBuilder().AddDocIDs(cvesToFilter...).ProtoQuery()
		}
		cloned = search.ConjunctionQuery(cloned, filter)
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
	if options.IncludeExploitData {
		cloned.Selects = append(cloned.Selects,
			search.NewQuerySelect(search.CVE).
				Distinct().
				AggrFunc(aggregatefunc.Count).
				Filter(knownExploitCountAlias, search.NewQueryBuilder().
					AddBools(search.CisaKev, true).ProtoQuery()).
				Proto(),
			search.NewQuerySelect(search.CVE).
				Distinct().
				AggrFunc(aggregatefunc.Count).
				Filter(knownRansomwareCountAlias, search.NewQueryBuilder().
					AddBools(search.KnownRansomwareCampaign, true).ProtoQuery()).
				Proto(),
		)
	}

	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{search.CVE.String()},
	}

	return cloned
}

// cveNameResponse is the minimal scan target for phase 1 when filtering by CVE name.
type cveNameResponse struct {
	CVE string `db:"cve"`
}

func (v *imageCVEFlatViewImpl) getFilteredCVEs(ctx context.Context, q *v1.Query, useCVENameFiltering bool) ([]string, error) {
	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	if useCVENameFiltering {
		var cveNames []string
		err := pgSearch.RunSelectRequestForSchemaFn[cveNameResponse](queryCtx, v.db, v.schema, withSelectCVENamesQuery(q), func(r *cveNameResponse) error {
			cveNames = append(cveNames, r.CVE)
			return nil
		})
		if err != nil {
			log.Error(err)
			return nil, err
		}
		return cveNames, nil
	}

	var cveIDsToFilter []string

	// TODO(@charmik) : Update the SQL query generator to not include 'ORDER BY' and 'GROUP BY' fields in the select clause (before where).
	//  SQL syntax does not need those fields in the select clause. The below query for example would work fine
	//  "SELECT JSONB_AGG(DISTINCT(image_cves.Id)) AS cve_id FROM image_cves GROUP BY image_cves.CveBaseInfo_Cve ORDER BY MAX(image_cves.Cvss) DESC LIMIT 20;"
	err := pgSearch.RunSelectRequestForSchemaFn[imageCVEFlatResponse](queryCtx, v.db, v.schema, withSelectCVEIdentifiersQuery(q), func(r *imageCVEFlatResponse) error {
		cveIDsToFilter = append(cveIDsToFilter, r.CVEIDs...)
		return nil
	})
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return cveIDsToFilter, nil
}
