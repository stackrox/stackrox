package rest

import (
	"context"
	"net/http"

	"github.com/stackrox/rox/central/views"
	vulnReqDS "github.com/stackrox/rox/central/vulnmgmt/vulnerabilityrequest/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

func (h *handler) listCVEs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query, err := parseSearchQuery(r)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	query = withPaginationLimits(query)
	if len(query.GetPagination().GetSortOptions()) == 0 {
		query.Pagination.SortOptions = []*v1.QuerySortOption{{Field: search.CVE.String()}}
	}

	// The REST list does not use row IDs or distro tuples afterwards, so it can use the
	// cheaper CVE-name based two-phase pagination.
	cveFlats, err := h.deps.CVEView.Get(ctx, query, views.ReadOptions{
		UseCVENameFiltering: true,
		IncludeExploitData:  true,
	})
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	cveNames := make([]string, 0, len(cveFlats))
	for _, cf := range cveFlats {
		cveNames = append(cveNames, cf.GetCVE())
	}
	exceptionCounts, err := batchExceptionCounts(ctx, h.deps.VulnReqStore, cveNames, exceptionStatusesForQuery(query))
	if err != nil {
		// Exception counts are supplementary; do not fail the list if they cannot be loaded.
		log.Errorf("failed to load exception counts: %v", err)
	}

	items := make([]CVEListItem, 0, len(cveFlats))
	for _, cf := range cveFlats {
		items = append(items, CVEListItem{
			CVE:                        cf.GetCVE(),
			TopCVSS:                    cf.GetTopCVSS(),
			TopNvdCVSS:                 cf.GetTopNVDCVSS(),
			TopEPSSProbability:         cf.GetEPSSProbability(),
			AffectedImageCount:         cf.GetAffectedImageCount(),
			FirstDiscoveredInSystem:    formatTime(cf.GetFirstDiscoveredInSystem()),
			PublishedOn:                formatTime(cf.GetPublishDate()),
			PendingExceptionCount:      exceptionCounts[cf.GetCVE()],
			KnownExploit:               cf.GetKnownExploit(),
			KnownRansomwareCampaignUse: cf.GetKnownRansomwareCampaignUse(),
		})
	}

	writeJSON(w, CVEListResponse{Items: items})
}

func batchExceptionCounts(ctx context.Context, store vulnReqDS.DataStore, cves []string, requestStatuses []string) (map[string]int32, error) {
	if store == nil || len(cves) == 0 {
		return nil, nil
	}
	cveSet := set.NewStringSet(cves...)
	qb := search.NewQueryBuilder().
		AddBools(search.ExpiredRequest, false).
		AddExactMatches(search.CVE, cves...)
	if len(requestStatuses) > 0 {
		qb.AddExactMatches(search.RequestStatus, requestStatuses...)
	}
	requests, err := store.SearchRawRequests(ctx, qb.ProtoQuery())
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
