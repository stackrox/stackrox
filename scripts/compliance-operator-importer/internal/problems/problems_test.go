package problems_test

import (
	"testing"

	"github.com/stackrox/co-acs-importer/internal/models"
	"github.com/stackrox/co-acs-importer/internal/problems"
)

// TestIMP_CLI_022_AddAndAllRoundtrip verifies that problems added to the Collector
// are returned verbatim by All(), preserving insertion order.
// Requirement: IMP-CLI-022 (problems[] entry appended for every problem).
func TestIMP_CLI_022_AddAndAllRoundtrip(t *testing.T) {
	c := problems.NewCollector()

	p1 := models.Problem{
		Severity:    models.SeverityError,
		Category:    models.CategoryAPI,
		ResourceRef: "ns/binding-a",
		Description: "API returned 503",
		FixHint:     "Retry later or check ACS endpoint.",
		Skipped:     true,
	}
	p2 := models.Problem{
		Severity:    models.SeverityWarning,
		Category:    models.CategoryConflict,
		ResourceRef: "ns/binding-b",
		Description: "Scan config already exists",
		FixHint:     "Delete or rename the existing config.",
		Skipped:     true,
	}

	c.Add(p1)
	c.Add(p2)

	got := c.All()
	if len(got) != 2 {
		t.Fatalf("expected 2 problems, got %d", len(got))
	}
	if got[0] != p1 {
		t.Errorf("first problem mismatch: got %+v, want %+v", got[0], p1)
	}
	if got[1] != p2 {
		t.Errorf("second problem mismatch: got %+v, want %+v", got[1], p2)
	}
}

// TestIMP_CLI_022_EmptyCollectorAllReturnsEmptySlice verifies that a fresh
// Collector returns an empty (non-nil) slice from All().
func TestIMP_CLI_022_EmptyCollectorAllReturnsEmptySlice(t *testing.T) {
	c := problems.NewCollector()
	got := c.All()
	if got == nil {
		t.Fatal("All() returned nil; want empty non-nil slice")
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 problems, got %d", len(got))
	}
}

// TestIMP_CLI_022_HasErrorsFalseWhenOnlyWarnings verifies that HasErrors returns
// false when only warning-severity problems are present.
// Requirement: IMP-CLI-022 (severity classification).
func TestIMP_CLI_022_HasErrorsFalseWhenOnlyWarnings(t *testing.T) {
	c := problems.NewCollector()
	c.Add(models.Problem{
		Severity:    models.SeverityWarning,
		Category:    models.CategoryConflict,
		ResourceRef: "ns/binding-c",
		Description: "Scan config already exists",
		FixHint:     "Delete the existing config and re-run.",
		Skipped:     true,
	})
	if c.HasErrors() {
		t.Error("HasErrors() returned true; expected false when only warnings present")
	}
}

// TestIMP_CLI_022_HasErrorsTrueWhenAnyErrorSeverity verifies that HasErrors
// returns true as soon as any error-severity problem is added.
// Requirement: IMP-CLI-022 (severity classification drives exit code logic).
func TestIMP_CLI_022_HasErrorsTrueWhenAnyErrorSeverity(t *testing.T) {
	c := problems.NewCollector()
	// Add a warning first to ensure we check all entries, not just the last.
	c.Add(models.Problem{
		Severity:    models.SeverityWarning,
		Category:    models.CategoryMapping,
		ResourceRef: "ns/binding-d",
		Description: "Schedule conversion warning",
		FixHint:     "Use a standard cron expression.",
		Skipped:     false,
	})
	c.Add(models.Problem{
		Severity:    models.SeverityError,
		Category:    models.CategoryAPI,
		ResourceRef: "ns/binding-e",
		Description: "API returned 400 Bad Request",
		FixHint:     "Check that the payload is valid and the cluster ID exists.",
		Skipped:     true,
	})
	if !c.HasErrors() {
		t.Error("HasErrors() returned false; expected true when error-severity problem is present")
	}
}

// TestIMP_CLI_022_HasErrorsFalseOnEmptyCollector verifies that an empty
// Collector reports no errors.
func TestIMP_CLI_022_HasErrorsFalseOnEmptyCollector(t *testing.T) {
	c := problems.NewCollector()
	if c.HasErrors() {
		t.Error("HasErrors() returned true on empty collector; expected false")
	}
}

// TestIMP_CLI_022_AllReturnsCopy verifies that mutations to the returned slice
// do not affect the Collector's internal state.
func TestIMP_CLI_022_AllReturnsCopy(t *testing.T) {
	c := problems.NewCollector()
	c.Add(models.Problem{
		Severity:    models.SeverityError,
		Category:    models.CategoryAPI,
		ResourceRef: "ns/binding-f",
		Description: "Transient API failure",
		FixHint:     "Increase --max-retries or check ACS health.",
		Skipped:     true,
	})

	got := c.All()
	got[0].Description = "mutated"

	// Second call must return the original value.
	fresh := c.All()
	if fresh[0].Description == "mutated" {
		t.Error("All() returned a reference to internal state; expected an independent copy")
	}
}
