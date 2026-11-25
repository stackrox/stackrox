package rest

import (
	"net/http"

	"github.com/stackrox/rox/central/views/common"
	"github.com/stackrox/rox/pkg/sync"
)

func (h *handler) getCounts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query, err := parseSearchQuery(r)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	countQuery := queryWithoutTraversal(query)

	// Each count walks a different entity table and can take seconds on large
	// deployments. Run them concurrently so the endpoint latency is the slowest
	// query instead of their sum.
	var (
		severityCounts  common.ResourceCountByCVESeverity
		imageCount      int
		deploymentCount int
		severityErr     error
		imageErr        error
		deploymentErr   error
		wg              sync.WaitGroup
	)
	wg.Add(3)
	go func() {
		defer wg.Done()
		severityCounts, severityErr = h.deps.CVECountView.CountBySeverity(ctx, countQuery.CloneVT())
	}()
	go func() {
		defer wg.Done()
		imageCount, imageErr = h.deps.ImageStore.Count(ctx, countQuery.CloneVT())
	}()
	go func() {
		defer wg.Done()
		deploymentCount, deploymentErr = h.deps.DeploymentStore.Count(ctx, countQuery.CloneVT())
	}()
	wg.Wait()

	if severityErr != nil {
		writeError(w, severityErr, http.StatusInternalServerError)
		return
	}
	if imageErr != nil {
		writeError(w, imageErr, http.StatusInternalServerError)
		return
	}
	if deploymentErr != nil {
		writeError(w, deploymentErr, http.StatusInternalServerError)
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
