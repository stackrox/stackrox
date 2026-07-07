package rest

import (
	"net/http"

	"github.com/stackrox/rox/pkg/search"
)

func (h *handler) getCVEDetail(w http.ResponseWriter, r *http.Request, cve string) {
	ctx := r.Context()

	q := search.NewQueryBuilder().AddExactMatches(search.CVE, cve).ProtoQuery()
	cves, err := h.cveDS.SearchRawImageCVEs(ctx, q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	seen := make(map[string]bool)
	var details []CVEDetailItem
	for _, c := range cves {
		info := c.GetCveBaseInfo()
		key := info.GetCve() + "|" + info.GetSummary()
		if seen[key] {
			continue
		}
		seen[key] = true
		details = append(details, CVEDetailItem{
			CVE:     info.GetCve(),
			Summary: info.GetSummary(),
			Link:    info.GetLink(),
		})
	}

	writeJSON(w, CVEDetailResponse{DistroDetails: details})
}
