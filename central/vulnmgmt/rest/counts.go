package rest

import (
	"net/http"
)

func (h *handler) getCounts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query, err := parseSearchQuery(r)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	countQuery := queryWithoutTraversal(query)

	severityCounts, err := h.deps.CVECountView.CountBySeverity(ctx, countQuery)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	imageCount, err := h.deps.ImageStore.Count(ctx, countQuery)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	deploymentCount, err := h.deps.DeploymentStore.Count(ctx, countQuery)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	severity := SeverityCounts{}
	if severityCounts != nil {
		severity = SeverityCounts{
			Critical:  severityCounts.GetCriticalSeverityCount().GetTotal(),
			Important: severityCounts.GetImportantSeverityCount().GetTotal(),
			Moderate:  severityCounts.GetModerateSeverityCount().GetTotal(),
			Low:       severityCounts.GetLowSeverityCount().GetTotal(),
			Unknown:   severityCounts.GetUnknownSeverityCount().GetTotal(),
		}
	}

	writeJSON(w, CountsResponse{
		CVE:        severity.Critical + severity.Important + severity.Moderate + severity.Low + severity.Unknown,
		Image:      imageCount,
		Deployment: deploymentCount,
		Severity:   severity,
	})
}
