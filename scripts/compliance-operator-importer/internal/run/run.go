// Package run orchestrates a full importer execution: CO discovery, ACS
// reconciliation, problem collection, report generation, and exit-code
// determination.
package run

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/stackrox/co-acs-importer/internal/cofetch"
	"github.com/stackrox/co-acs-importer/internal/mapping"
	"github.com/stackrox/co-acs-importer/internal/models"
	"github.com/stackrox/co-acs-importer/internal/problems"
	"github.com/stackrox/co-acs-importer/internal/reconcile"
	"github.com/stackrox/co-acs-importer/internal/report"
)

// Exit code constants (IMP-CLI-017..019, IMP-ERR-003).
const (
	ExitSuccess      = 0 // all bindings processed without failures
	ExitFatalError   = 1 // preflight/config failure; no import attempted
	ExitPartialError = 2 // at least one binding failed
)

// Runner orchestrates the full import run.
type Runner struct {
	cfg       *models.Config
	acsClient models.ACSClient
	coClient  cofetch.COClient
	out       io.Writer // injectable; defaults to os.Stdout
}

// NewRunner creates a Runner ready to execute, writing console output to os.Stdout.
func NewRunner(cfg *models.Config, acsClient models.ACSClient, coClient cofetch.COClient) *Runner {
	return &Runner{
		cfg:       cfg,
		acsClient: acsClient,
		coClient:  coClient,
		out:       os.Stdout,
	}
}

// WithOutput returns a shallow copy of the Runner writing console output to w.
// Intended for tests that need to capture or suppress printed output.
func (r *Runner) WithOutput(w io.Writer) *Runner {
	cp := *r
	cp.out = w
	return &cp
}

// printf is a convenience wrapper so callers don't need to handle format errors.
func (r *Runner) printf(format string, args ...interface{}) {
	fmt.Fprintf(r.out, format, args...) //nolint:errcheck // best-effort console output
}

// Run executes the full import and returns the appropriate exit code.
//
// Execution steps:
//  1. List existing ACS scan config names to build the existingNames set.
//  2. List ScanSettingBindings from the CO source cluster.
//  3. For each binding: fetch its ScanSetting, build the ACS payload, reconcile.
//  4. Collect all problems and build the final Report.
//  5. Optionally write the JSON report to --report-json path.
//  6. Print the console summary (IMP-CLI-020).
//  7. Return exit code 0, 1, or 2 (IMP-CLI-017..019, IMP-ERR-003).
func (r *Runner) Run(ctx context.Context) int {
	collector := problems.NewCollector()
	builder := report.NewBuilder(r.cfg)

	// Step 1: list existing ACS scan configs to populate the deduplication set.
	// Failure here is fatal (IMP-CLI-018): we cannot safely proceed without
	// knowing which names already exist.
	summaries, err := r.acsClient.ListScanConfigurations(ctx)
	if err != nil {
		r.printf("FATAL: failed to list ACS scan configurations: %v\n", err)
		return ExitFatalError
	}
	existingNames := make(map[string]bool, len(summaries))
	for _, s := range summaries {
		existingNames[s.ScanName] = true
	}

	// Step 2: discover CO ScanSettingBindings.
	// Failure here is also fatal (IMP-CLI-018).
	bindings, err := r.coClient.ListScanSettingBindings(ctx)
	if err != nil {
		r.printf("FATAL: failed to list ScanSettingBindings: %v\n", err)
		return ExitFatalError
	}

	// maxRetries defaults to 1 (single attempt) when cfg.MaxRetries is zero.
	maxRetries := r.cfg.MaxRetries
	if maxRetries < 1 {
		maxRetries = 1
	}
	rec := reconcile.NewReconciler(r.acsClient, maxRetries, r.cfg.DryRun)

	// Step 3: process each binding independently.
	// Per-binding failures skip that binding and record a problem; other bindings
	// continue processing (IMP-CLI-022, IMP-MAP-011).
	for _, binding := range bindings {
		r.processBinding(ctx, binding, existingNames, rec, collector, builder)
	}

	// Step 4: build the final report.
	finalReport := builder.Build(collector.All())

	// Step 5: write JSON report when requested (IMP-CLI-021).
	if r.cfg.ReportJSON != "" {
		if err := builder.WriteJSON(r.cfg.ReportJSON, finalReport); err != nil {
			r.printf("WARNING: failed to write JSON report to %q: %v\n", r.cfg.ReportJSON, err)
		}
	}

	// Step 6: print console summary (IMP-CLI-020).
	r.printSummary(finalReport)

	// Step 7: determine exit code (IMP-CLI-017..019, IMP-ERR-003).
	if finalReport.Counts.Failed > 0 || collector.HasErrors() {
		return ExitPartialError // IMP-CLI-019
	}
	return ExitSuccess // IMP-CLI-017
}

// processBinding handles a single ScanSettingBinding: fetches its ScanSetting,
// maps it to an ACS payload, and calls the reconciler. All failures are recorded
// as problems and do not abort processing of remaining bindings.
func (r *Runner) processBinding(
	ctx context.Context,
	binding cofetch.ScanSettingBinding,
	existingNames map[string]bool,
	rec *reconcile.Reconciler,
	collector *problems.Collector,
	builder *report.Builder,
) {
	// Derive a stable resource reference for problem entries.
	resourceRef := fmt.Sprintf("%s/%s", binding.Namespace, binding.Name)

	// Build the source for ReportItem entries.
	source := models.ReportItemSource{
		Namespace:       binding.Namespace,
		BindingName:     binding.Name,
		ScanSettingName: binding.ScanSettingName,
	}

	// Fetch the referenced ScanSetting (IMP-MAP-008..010).
	ss, err := r.coClient.GetScanSetting(ctx, binding.Namespace, binding.ScanSettingName)
	if err != nil {
		collector.Add(models.Problem{
			Severity:    models.SeverityError,
			Category:    models.CategoryInput,
			ResourceRef: resourceRef,
			Description: fmt.Sprintf("ScanSetting %q referenced by binding %q could not be fetched: %v", binding.ScanSettingName, binding.Name, err),
			FixHint:     fmt.Sprintf("Ensure ScanSetting %q exists in namespace %q and the importer service account has read access.", binding.ScanSettingName, binding.Namespace),
			Skipped:     true,
		})
		builder.RecordItem(models.ReportItem{
			Source: source,
			Action: "fail",
			Reason: "ScanSetting not found",
			Error:  err.Error(),
		})
		return
	}

	// Map the CO resources to an ACS create payload (IMP-MAP-001..015).
	result := mapping.MapBinding(binding, ss, r.cfg)
	if result.Problem != nil {
		// IMP-MAP-012..015: mapping problem => skip + record.
		collector.Add(*result.Problem)
		builder.RecordItem(models.ReportItem{
			Source: source,
			Action: "fail",
			Reason: "mapping error",
			Error:  result.Problem.Description,
		})
		return
	}

	// Reconcile: create or skip (IMP-IDEM-001..007, IMP-ERR-001..004).
	action := rec.Apply(ctx, *result.Payload, source, existingNames)

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
}

// printSummary writes the console summary to the configured output (IMP-CLI-020).
func (r *Runner) printSummary(rep models.Report) {
	dryRunLabel := "no"
	if r.cfg.DryRun {
		dryRunLabel = "yes"
	}
	r.printf("CO->ACS importer summary [dry-run: %s]:\n", dryRunLabel)
	r.printf("  Discovered:  %d bindings\n", rep.Counts.Discovered)
	r.printf("  Created:     %d\n", rep.Counts.Create)
	r.printf("  Skipped:     %d\n", rep.Counts.Skip)
	r.printf("  Failed:      %d\n", rep.Counts.Failed)
}
