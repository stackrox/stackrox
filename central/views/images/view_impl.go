package images

import (
	"context"

	"github.com/stackrox/rox/central/views/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

var (
	queryTimeout = env.PostgresVMStatementTimeout.DurationSetting()
)

type imageCoreViewImpl struct {
	schema *walker.Schema
	db     postgres.DB
}

func (v *imageCoreViewImpl) Get(ctx context.Context, query *v1.Query) ([]ImageCore, error) {
	if err := common.ValidateQuery(query); err != nil {
		return nil, err
	}

	query = withSelectQuery(query)

	queryCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(ctx, queryTimeout)
	defer cancel()

	ret := make([]ImageCore, 0, paginated.GetLimit(query.GetPagination().GetLimit(), 100))
	err := pgSearch.RunSelectRequestForSchemaFn[imageResponse](queryCtx, v.db, v.schema, query, func(r *imageResponse) error {
		ret = append(ret, r)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// SeverityRankSortField is a virtual sort field the UI can request to order images by severity.
// It expands into a lexicographic ordering (critical first) across the denormalized, per-image
// scanstats_*cvecount columns on images_v2, instead of computing a per-request count(distinct)
// aggregate over image_cves_v2. The stored counts are unfiltered totals across all vulnerability
// states; which images appear is still gated by the query's own filters (e.g. State:OBSERVED).
const SeverityRankSortField = "Severity Rank"

// severityRankStoredCountFields are the stored per-severity count fields that SeverityRankSortField
// expands into, ordered from most to least severe.
var severityRankStoredCountFields = []search.FieldLabel{
	search.CriticalCVECount,
	search.ImportantCVECount,
	search.ModerateCVECount,
	search.LowCVECount,
	search.UnknownCVECount,
}

// expandSeverityRankSort replaces a SeverityRankSortField sort option with the ordered set of stored
// per-severity count sorts (preserving the requested direction) followed by a unique tiebreaker so
// LIMIT/OFFSET pagination stays deterministic across pages.
func expandSeverityRankSort(q *v1.Query, tiebreaker search.FieldLabel) {
	sortOptions := q.GetPagination().GetSortOptions()
	idx := -1
	for i, sortOption := range sortOptions {
		if sortOption.GetField() == SeverityRankSortField {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}

	reversed := sortOptions[idx].GetReversed()
	expanded := make([]*v1.QuerySortOption, 0, len(sortOptions)+len(severityRankStoredCountFields))
	expanded = append(expanded, sortOptions[:idx]...)
	for _, field := range severityRankStoredCountFields {
		expanded = append(expanded, &v1.QuerySortOption{Field: field.String(), Reversed: reversed})
	}
	expanded = append(expanded, &v1.QuerySortOption{Field: tiebreaker.String()})
	expanded = append(expanded, sortOptions[idx+1:]...)
	q.Pagination.SortOptions = expanded
}

func withSelectQuery(query *v1.Query) *v1.Query {
	cloned := query.CloneVT()
	var searchField search.FieldLabel
	if features.FlattenImageData.Enabled() {
		searchField = search.ImageID
	} else {
		searchField = search.ImageSHA
	}
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(searchField).Proto(),
	}

	// Expand the virtual "Severity Rank" sort into stored-column sorts before the severity-count
	// detection below, so it takes the cheap stored-column path rather than the count(distinct) one.
	expandSeverityRankSort(cloned, searchField)

	if common.IsSortBySeverityCounts(cloned) {
		cloned.GroupBy = &v1.QueryGroupBy{
			Fields: []string{searchField.String()},
		}
		cloned.Selects = append(cloned.Selects,
			common.WithCountBySeverityAndFixabilityQuery(query, search.CVE).GetSelects()...,
		)
	}
	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{searchField.String()},
	}

	// This is to minimize UI change and hide an implementation detail that the query groups images by their SHA.
	// Because of this, for a field that is not in images table, there can be multiple values of that field per SHA.
	// So in order to sort by that field, we need some kind of aggregate applied to it.
	for _, sortOption := range cloned.GetPagination().GetSortOptions() {
		if sortOption.GetField() == search.Severity.String() {
			sortOption.Field = search.SeverityMax.String()
		}
		if sortOption.GetField() == search.CVSS.String() {
			sortOption.Field = search.CVSSMax.String()
		}
		if sortOption.GetField() == search.NVDCVSS.String() {
			sortOption.Field = search.NVDCVSSMax.String()
		}
		if sortOption.GetField() == search.OperatingSystem.String() {
			// Both 'Operating System' in CVE and 'Image OS' in an image containing that CVE have the same value.
			// Don't need an aggregate here since 'Image OS' is in images schema
			sortOption.Field = search.ImageOS.String()
		}
	}

	return cloned
}
