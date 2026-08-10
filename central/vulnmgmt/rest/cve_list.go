package rest

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/stackrox/rox/central/views"
	vulnReqDataStore "github.com/stackrox/rox/central/vulnmgmt/vulnerabilityrequest/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/parser"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	"github.com/stackrox/rox/pkg/set"
)

func (h *handler) listCVEs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query, _, err := parser.ParseURLQuery(r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	addSeveritySortTiebreaker(query)

	cveFlats, err := h.cveView.Get(ctx, query, views.ReadOptions{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	countQuery := query.CloneVT()
	countQuery.Pagination = nil
	totalCount, err := h.cveView.Count(ctx, countQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cveNames := make([]string, len(cveFlats))
	for i, cf := range cveFlats {
		cveNames[i] = cf.GetCVE()
	}
	var exceptionCounts map[string]int32
	if h.vulnReqStore != nil {
		exceptionCounts, _ = batchExceptionCounts(ctx, h.vulnReqStore, cveNames)
	}

	items := make([]CVEListItem, len(cveFlats))
	for i, cf := range cveFlats {
		items[i] = CVEListItem{
			CVE:                     cf.GetCVE(),
			TopSeverity:             cf.GetSeverity().String(),
			TopCVSS:                 cf.GetTopCVSS(),
			TopNvdCVSS:              cf.GetTopNVDCVSS(),
			TopEPSSProbability:      cf.GetEPSSProbability(),
			AffectedImageCount:      cf.GetAffectedImageCount(),
			FirstDiscoveredInSystem: formatTime(cf.GetFirstDiscoveredInSystem()),
			PublishedOn:             formatTime(cf.GetPublishDate()),
			PendingExceptionCount:   exceptionCounts[cf.GetCVE()],
		}
	}

	writeJSON(w, CVEListResponse{Items: items, TotalCount: totalCount})
}

func batchExceptionCounts(ctx context.Context, store vulnReqDataStore.DataStore, cves []string) (map[string]int32, error) {
	if len(cves) == 0 {
		return nil, nil
	}
	cveSet := set.NewStringSet(cves...)
	q := search.NewQueryBuilder().
		AddBools(search.ExpiredRequest, false).
		AddExactMatches(search.CVE, cves...).
		ProtoQuery()
	requests, err := store.SearchRawRequests(ctx, q)
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int32, len(cves))
	for _, req := range requests {
		for _, cve := range req.GetCves().GetCves() {
			if cveSet.Contains(cve) {
				counts[cve]++
			}
		}
	}
	return counts, nil
}

func addSeveritySortTiebreaker(query *v1.Query) {
	pagination := query.GetPagination()
	if pagination == nil {
		return
	}
	imageField := search.ImageSHA
	if features.FlattenImageData.Enabled() {
		imageField = search.ImageID
	}
	for _, so := range pagination.GetSortOptions() {
		if strings.EqualFold(so.GetField(), search.Severity.String()) {
			pagination.SortOptions = append(pagination.SortOptions, &v1.QuerySortOption{
				Field:    imageField.String(),
				Reversed: so.GetReversed(),
				AggregateBy: &v1.AggregateBy{
					AggrFunc: aggregatefunc.Count.Proto(),
					Distinct: true,
				},
			})
			return
		}
	}
}

func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}
