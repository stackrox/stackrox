package run

import (
	"context"
	"fmt"

	"github.com/stackrox/co-acs-importer/internal/adopt"
	"github.com/stackrox/co-acs-importer/internal/cofetch"
	"github.com/stackrox/co-acs-importer/internal/mapping"
	"github.com/stackrox/co-acs-importer/internal/merge"
	"github.com/stackrox/co-acs-importer/internal/models"
	"github.com/stackrox/co-acs-importer/internal/problems"
	"github.com/stackrox/co-acs-importer/internal/reconcile"
	"github.com/stackrox/co-acs-importer/internal/report"
)

// RunMultiCluster executes the importer in multi-cluster mode.
//
// Steps:
//  1. List existing ACS scan configs to build the existingNames set.
//  2. For each cluster source:
//     a. List ScanSettingBindings.
//     b. Map each SSB to an ACS payload, using the cluster's ACS ID.
//  3. Merge SSBs across clusters by name.
//  4. Reconcile merged payloads against ACS.
//  5. Build and write report.
//  6. Print console summary.
//  7. Return exit code.
func (r *Runner) RunMultiCluster(ctx context.Context, sources []ClusterSource) int {
	collector := problems.NewCollector()
	builder := report.NewBuilder(r.cfg)

	// Step 1: list existing ACS scan configs.
	r.status.Stage("Inventory", "listing existing ACS scan configurations")
	summaries, err := r.acsClient.ListScanConfigurations(ctx)
	if err != nil {
		r.status.Failf("failed to list ACS scan configurations: %v", err)
		return ExitFatalError
	}
	existingNames := make(map[string]string, len(summaries))
	for _, s := range summaries {
		existingNames[s.ScanName] = s.ID
	}
	r.status.OKf("found %d existing scan configurations", len(summaries))

	// ssbClusterInfo tracks per-SSB per-cluster metadata needed for adoption.
	type ssbClusterInfo struct {
		namespace     string
		oldSettingRef string
		clusterLabel  string
		coClient      cofetch.COClient
	}
	// Key: SSB name, value: list of cluster infos (one per cluster that has the SSB).
	ssbAdoptionMap := make(map[string][]ssbClusterInfo)

	// Step 2: collect SSBs from all clusters and map them.
	clusterSSBs := make(map[string][]merge.MappedSSB)

	for _, source := range sources {
		r.status.Stagef("Scan", "cluster %s (ACS ID: %s)", source.Label, source.ACSClusterID)

		bindings, err := source.COClient.ListScanSettingBindings(ctx)
		if err != nil {
			r.status.Warnf("failed to list ScanSettingBindings from %s: %v", source.Label, err)
			collector.Add(models.Problem{
				Severity:    models.SeverityError,
				Category:    models.CategoryInput,
				ResourceRef: source.Label,
				Description: fmt.Sprintf("Failed to list ScanSettingBindings from cluster %q: %v", source.Label, err),
				FixHint:     "Check cluster connectivity and permissions.",
				Skipped:     true,
			})
			continue
		}

		r.status.OKf("found %d ScanSettingBindings", len(bindings))

		for _, binding := range bindings {
			// Fetch the ScanSetting.
			ss, err := source.COClient.GetScanSetting(ctx, binding.Namespace, binding.ScanSettingName)
			if err != nil {
				collector.Add(models.Problem{
					Severity:    models.SeverityError,
					Category:    models.CategoryInput,
					ResourceRef: fmt.Sprintf("%s:%s/%s", source.Label, binding.Namespace, binding.Name),
					Description: fmt.Sprintf("ScanSetting %q referenced by binding %q in cluster %q could not be fetched: %v", binding.ScanSettingName, binding.Name, source.Label, err),
					FixHint:     fmt.Sprintf("Ensure ScanSetting %q exists in namespace %q on cluster %q.", binding.ScanSettingName, binding.Namespace, source.Label),
					Skipped:     true,
				})
				continue
			}

			// Map the binding to an ACS payload.
			// Create a temporary config with the cluster ID for this source.
			tempCfg := *r.cfg
			tempCfg.ACSClusterID = source.ACSClusterID

			result := mapping.MapBinding(binding, ss, &tempCfg)
			if result.Problem != nil {
				collector.Add(*result.Problem)
				continue
			}

			// Track metadata for adoption.
			ssbAdoptionMap[binding.Name] = append(ssbAdoptionMap[binding.Name], ssbClusterInfo{
				namespace:     binding.Namespace,
				oldSettingRef: binding.ScanSettingName,
				clusterLabel:  source.Label,
				coClient:      source.COClient,
			})

			// Add to the cluster's SSB list for merging.
			clusterSSBs[source.ACSClusterID] = append(clusterSSBs[source.ACSClusterID], merge.MappedSSB{
				Name:     binding.Name,
				Profiles: extractProfileNames(binding),
				Payload:  *result.Payload,
			})
		}
	}

	// Step 3: merge SSBs across clusters.
	r.status.Stage("Merge", "combining ScanSettingBindings across clusters")
	mergeResult := merge.MergeSSBs(clusterSSBs)

	for _, problem := range mergeResult.Problems {
		collector.Add(problem)
		r.status.Warnf("%s: %s", problem.ResourceRef, problem.Description)
	}

	r.status.OKf("merged into %d unique scan configurations", len(mergeResult.Merged))

	// Step 4: reconcile merged payloads.
	r.status.Stage("Reconcile", "applying scan configurations to ACS")
	maxRetries := r.cfg.MaxRetries
	if maxRetries < 1 {
		maxRetries = 1
	}
	rec := reconcile.NewReconciler(r.acsClient, maxRetries, r.cfg.DryRun, r.cfg.OverwriteExisting)

	var adoptRequests []adopt.Request

	for _, merged := range mergeResult.Merged {
		source := models.ReportItemSource{
			BindingName: merged.Name,
			// For multi-cluster, namespace and scanSettingName are per-cluster, so we leave them generic.
			Namespace:       "multi-cluster",
			ScanSettingName: "merged",
		}

		action := rec.Apply(ctx, merged.Payload, source, existingNames)

		switch action.ActionType {
		case "create":
			r.status.OKf("%s → created (%d clusters)", merged.Name, len(merged.Payload.Clusters))
		case "update":
			r.status.OKf("%s → updated (%d clusters)", merged.Name, len(merged.Payload.Clusters))
		case "skip":
			r.status.Detailf("%s → skipped (already exists)", merged.Name)
		case "fail":
			r.status.Failf("%s → %s", merged.Name, action.Reason)
		}

		item := models.ReportItem{
			Source:          action.Source,
			Action:          action.ActionType,
			Reason:          action.Reason,
			Attempts:        action.Attempts,
			ACSScanConfigID: action.ACSScanConfigID,
		}
		if action.Err != nil {
			item.Error = action.Err.Error()
		}
		builder.RecordItem(item)

		if action.Problem != nil {
			collector.Add(*action.Problem)
		}

		// Collect adoption requests for successfully created scan configs.
		if action.ActionType == "create" && !r.cfg.DryRun {
			for _, info := range ssbAdoptionMap[merged.Name] {
				adoptRequests = append(adoptRequests, adopt.Request{
					SSBName:       merged.Name,
					SSBNamespace:  info.namespace,
					OldSettingRef: info.oldSettingRef,
					ClusterLabel:  info.clusterLabel,
					COClient:      info.coClient,
				})
			}
		}
	}

	// Step 4b: adopt SSBs whose scan configs were just created.
	if len(adoptRequests) > 0 {
		r.runAdoption(ctx, adoptRequests)
	}

	// Step 5: build and write report.
	finalReport := builder.Build(collector.All())

	if r.cfg.ReportJSON != "" {
		r.status.Stage("Report", "writing JSON report")
		if err := builder.WriteJSON(r.cfg.ReportJSON, finalReport); err != nil {
			r.status.Warnf("failed to write JSON report to %q: %v", r.cfg.ReportJSON, err)
		} else {
			r.status.OKf("report written to %s", r.cfg.ReportJSON)
		}
	}

	// Step 6: print console summary.
	r.printf("\n")
	r.printSummary(finalReport)

	// Step 7: determine exit code.
	if finalReport.Counts.Failed > 0 || collector.HasErrors() {
		return ExitPartialError
	}
	return ExitSuccess
}

// extractProfileNames extracts profile names from a binding.
func extractProfileNames(binding cofetch.ScanSettingBinding) []string {
	var names []string
	for _, p := range binding.Profiles {
		names = append(names, p.Name)
	}
	return names
}
