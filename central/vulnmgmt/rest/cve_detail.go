package rest

import (
	"net/http"
	"sort"
	"strings"

	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

// distroPriorityMap mirrors the UI's distroPriorityMap in sortUtils.tsx.
var distroPriorityMap = map[string]int{
	"rhel":   1,
	"centos": 2,
	"ubuntu": 3,
	"debian": 4,
	"alpine": 5,
	"amzn":   6,
}

var distroPriorityOther = 7

func distroPriority(operatingSystem string) int {
	for prefix, priority := range distroPriorityMap {
		if strings.HasPrefix(operatingSystem, prefix) {
			return priority
		}
	}
	return distroPriorityOther
}

func (h *handler) getCVEDetail(w http.ResponseWriter, r *http.Request, cve string) {
	ctx := r.Context()

	query, err := parseSearchQuery(r)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	query = search.ConjunctionQuery(
		queryWithoutTraversal(query),
		search.NewQueryBuilder().AddExactMatches(search.CVE, cve).ProtoQuery(),
	)

	cves, err := h.deps.CVEDataStore.SearchRawImageCVEs(ctx, query)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	imageIDs := set.NewStringSet()
	for _, c := range cves {
		if imageID := c.GetImageIdV2(); imageID != "" {
			imageIDs.Add(imageID)
		}
	}
	operatingSystems := make(map[string]string, imageIDs.Cardinality())
	if imageIDs.Cardinality() > 0 {
		images, err := h.deps.ImageStore.GetManyImageMetadata(ctx, imageIDs.AsSlice())
		if err != nil {
			writeError(w, err, http.StatusInternalServerError)
			return
		}
		for _, img := range images {
			operatingSystems[img.GetId()] = img.GetScan().GetOperatingSystem()
		}
	}

	seen := make(map[string]bool)
	details := make([]CVEDetailItem, 0, len(cves))
	for _, c := range cves {
		info := c.GetCveBaseInfo()
		key := info.GetCve() + "|" + info.GetSummary()
		if seen[key] {
			continue
		}
		seen[key] = true
		details = append(details, CVEDetailItem{
			CVE:             info.GetCve(),
			Summary:         info.GetSummary(),
			Link:            info.GetLink(),
			OperatingSystem: operatingSystems[c.GetImageIdV2()],
		})
	}

	sort.SliceStable(details, func(i, j int) bool {
		pi, pj := distroPriority(details[i].OperatingSystem), distroPriority(details[j].OperatingSystem)
		if pi != pj {
			return pi < pj
		}
		return details[i].OperatingSystem < details[j].OperatingSystem
	})

	writeJSON(w, CVEDetailResponse{DistroDetails: details})
}
