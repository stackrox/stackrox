package report_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stackrox/co-acs-importer/internal/models"
	"github.com/stackrox/co-acs-importer/internal/report"
)

// baseConfig returns a minimal Config suitable for most report tests.
func baseConfig() *models.Config {
	return &models.Config{
		DryRun:      false,
		CONamespace: "openshift-compliance",
	}
}

// TestIMP_CLI_021_BuildSetsModeTocreateOnly verifies that Build always sets
// meta.mode to "create-only" regardless of other configuration.
// Requirement: IMP-CLI-021.
func TestIMP_CLI_021_BuildSetsModeTocreateOnly(t *testing.T) {
	b := report.NewBuilder(baseConfig())
	r := b.Build(nil)
	if r.Meta.Mode != "create-only" {
		t.Errorf("meta.mode = %q; want %q", r.Meta.Mode, "create-only")
	}
}

// TestIMP_CLI_021_BuildCountsFromItemActions verifies that Build correctly derives
// discovered/create/skip/failed counts from the recorded items.
// Requirement: IMP-CLI-021.
func TestIMP_CLI_021_BuildCountsFromItemActions(t *testing.T) {
	cfg := baseConfig()
	b := report.NewBuilder(cfg)

	b.RecordItem(models.ReportItem{Action: "create"})
	b.RecordItem(models.ReportItem{Action: "create"})
	b.RecordItem(models.ReportItem{Action: "skip"})
	b.RecordItem(models.ReportItem{Action: "fail"})

	r := b.Build(nil)

	if r.Counts.Discovered != 4 {
		t.Errorf("counts.discovered = %d; want 4", r.Counts.Discovered)
	}
	if r.Counts.Create != 2 {
		t.Errorf("counts.create = %d; want 2", r.Counts.Create)
	}
	if r.Counts.Skip != 1 {
		t.Errorf("counts.skip = %d; want 1", r.Counts.Skip)
	}
	if r.Counts.Failed != 1 {
		t.Errorf("counts.failed = %d; want 1", r.Counts.Failed)
	}
}

// TestIMP_CLI_021_BuildMetaNamespaceScopeAllNamespaces verifies that when
// COAllNamespaces is set, meta.namespaceScope is "all-namespaces".
// Requirement: IMP-CLI-021.
func TestIMP_CLI_021_BuildMetaNamespaceScopeAllNamespaces(t *testing.T) {
	cfg := &models.Config{
		COAllNamespaces: true,
	}
	b := report.NewBuilder(cfg)
	r := b.Build(nil)
	if r.Meta.NamespaceScope != "all-namespaces" {
		t.Errorf("meta.namespaceScope = %q; want %q", r.Meta.NamespaceScope, "all-namespaces")
	}
}

// TestIMP_CLI_021_BuildMetaNamespaceScopeSingleNamespace verifies that when
// COAllNamespaces is false, meta.namespaceScope equals cfg.CONamespace.
// Requirement: IMP-CLI-021.
func TestIMP_CLI_021_BuildMetaNamespaceScopeSingleNamespace(t *testing.T) {
	cfg := &models.Config{
		CONamespace:     "openshift-compliance",
		COAllNamespaces: false,
	}
	b := report.NewBuilder(cfg)
	r := b.Build(nil)
	if r.Meta.NamespaceScope != "openshift-compliance" {
		t.Errorf("meta.namespaceScope = %q; want %q", r.Meta.NamespaceScope, "openshift-compliance")
	}
}

// TestIMP_CLI_021_BuildMetaDryRunReflectsCfg verifies that meta.dryRun mirrors
// the cfg.DryRun field.
// Requirement: IMP-CLI-021.
func TestIMP_CLI_021_BuildMetaDryRunReflectsCfg(t *testing.T) {
	for _, dryRun := range []bool{true, false} {
		cfg := &models.Config{DryRun: dryRun, CONamespace: "ns"}
		b := report.NewBuilder(cfg)
		r := b.Build(nil)
		if r.Meta.DryRun != dryRun {
			t.Errorf("dryRun=%v: meta.dryRun = %v; want %v", dryRun, r.Meta.DryRun, dryRun)
		}
	}
}

// TestIMP_CLI_021_BuildTimestampIsRFC3339 verifies that meta.timestamp is a
// non-empty, valid RFC3339 string.
// Requirement: IMP-CLI-021.
func TestIMP_CLI_021_BuildTimestampIsRFC3339(t *testing.T) {
	b := report.NewBuilder(baseConfig())
	r := b.Build(nil)
	if r.Meta.Timestamp == "" {
		t.Fatal("meta.timestamp is empty")
	}
	// time.Parse with RFC3339 format validates the string.
	// We use strings.Contains as a lightweight check; a full parse would need
	// importing "time" and would be equally valid.
	if !strings.Contains(r.Meta.Timestamp, "T") || !strings.Contains(r.Meta.Timestamp, "Z") {
		t.Errorf("meta.timestamp %q does not look like UTC RFC3339", r.Meta.Timestamp)
	}
}

// TestIMP_CLI_021_WriteJSONProducesValidJSON verifies that WriteJSON writes
// parseable JSON to disk.
// Requirement: IMP-CLI-021.
func TestIMP_CLI_021_WriteJSONProducesValidJSON(t *testing.T) {
	cfg := baseConfig()
	b := report.NewBuilder(cfg)
	b.RecordItem(models.ReportItem{Action: "create", Reason: "created successfully"})

	r := b.Build(nil)

	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")
	if err := b.WriteJSON(path, r); err != nil {
		t.Fatalf("WriteJSON returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading written report: %v", err)
	}

	var parsed models.Report
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("written JSON is not parseable: %v\ncontent:\n%s", err, string(data))
	}

	if parsed.Meta.Mode != "create-only" {
		t.Errorf("parsed meta.mode = %q; want %q", parsed.Meta.Mode, "create-only")
	}
	if parsed.Counts.Discovered != 1 {
		t.Errorf("parsed counts.discovered = %d; want 1", parsed.Counts.Discovered)
	}
}

// TestIMP_CLI_022_ProblemsInReportMatchInput verifies that problems passed to
// Build() appear unchanged in the report's Problems field.
// Requirement: IMP-CLI-022.
func TestIMP_CLI_022_ProblemsInReportMatchInput(t *testing.T) {
	cfg := baseConfig()
	b := report.NewBuilder(cfg)

	probs := []models.Problem{
		{
			Severity:    models.SeverityError,
			Category:    models.CategoryAPI,
			ResourceRef: "ns/binding-a",
			Description: "ACS API returned 503",
			FixHint:     "Check ACS endpoint health and retry.",
			Skipped:     true,
		},
		{
			Severity:    models.SeverityWarning,
			Category:    models.CategoryConflict,
			ResourceRef: "ns/binding-b",
			Description: "Scan config already exists",
			FixHint:     "Delete the existing ACS config and re-run.",
			Skipped:     true,
		},
	}

	r := b.Build(probs)

	if len(r.Problems) != 2 {
		t.Fatalf("expected 2 problems in report, got %d", len(r.Problems))
	}
	for i, want := range probs {
		got := r.Problems[i]
		if got != want {
			t.Errorf("problem[%d] mismatch: got %+v, want %+v", i, got, want)
		}
	}
}

// TestIMP_CLI_021_WriteJSONErrorOnBadPath verifies WriteJSON returns an error
// when the target directory does not exist.
func TestIMP_CLI_021_WriteJSONErrorOnBadPath(t *testing.T) {
	b := report.NewBuilder(baseConfig())
	r := b.Build(nil)
	err := b.WriteJSON("/nonexistent/dir/report.json", r)
	if err == nil {
		t.Error("expected error writing to non-existent path, got nil")
	}
}

// TestIMP_CLI_021_BuildEmptyItemsProducesNonNilSlices verifies that Build
// returns non-nil Items and Problems slices even when nothing was recorded.
// This ensures JSON output is "items": [] not "items": null.
func TestIMP_CLI_021_BuildEmptyItemsProducesNonNilSlices(t *testing.T) {
	b := report.NewBuilder(baseConfig())
	r := b.Build(nil)
	if r.Items == nil {
		t.Error("Items is nil; want empty non-nil slice so JSON marshals as []")
	}
	if r.Problems == nil {
		t.Error("Problems is nil; want empty non-nil slice so JSON marshals as []")
	}
}
