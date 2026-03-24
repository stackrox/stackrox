// Package report assembles the final Report from accumulated run items and writes
// it to disk as indented JSON when --report-json is set.
package report

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/stackrox/co-acs-importer/internal/models"
)

// Builder accumulates per-binding ReportItems during a run and produces the
// final Report once all bindings have been processed.
type Builder struct {
	cfg   *models.Config
	items []models.ReportItem
}

// NewBuilder returns a Builder configured from cfg.
func NewBuilder(cfg *models.Config) *Builder {
	return &Builder{cfg: cfg}
}

// RecordItem appends a single binding outcome to the builder.
func (b *Builder) RecordItem(item models.ReportItem) {
	b.items = append(b.items, item)
}

// Build constructs the final Report from all recorded items and the supplied
// problems list.
//
// IMP-CLI-021: sets meta.mode = "create-only", meta.timestamp to current UTC
// RFC3339, meta.dryRun from cfg, meta.namespaceScope from cfg.
// IMP-CLI-021: computes counts from items actions.
func (b *Builder) Build(problems []models.Problem) models.Report {
	meta := models.ReportMeta{
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		DryRun:         b.cfg.DryRun,
		NamespaceScope: namespaceScope(b.cfg),
		Mode:           "create-only",
	}

	counts := models.ReportCounts{
		Discovered: len(b.items),
	}
	for _, it := range b.items {
		switch it.Action {
		case "create":
			counts.Create++
		case "skip":
			counts.Skip++
		case "fail":
			counts.Failed++
		}
	}

	items := b.items
	if items == nil {
		items = []models.ReportItem{}
	}
	if problems == nil {
		problems = []models.Problem{}
	}

	return models.Report{
		Meta:     meta,
		Counts:   counts,
		Items:    items,
		Problems: problems,
	}
}

// WriteJSON writes report as indented JSON to path.
// Returns an error if the file cannot be created or written.
// IMP-CLI-021: output must be valid, parseable JSON.
func (b *Builder) WriteJSON(path string, report models.Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report to JSON: %w", err)
	}
	// Append a trailing newline for POSIX text-file compliance.
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write report JSON to %q: %w", path, err)
	}
	return nil
}

// namespaceScope derives the namespaceScope string from cfg.
func namespaceScope(cfg *models.Config) string {
	if cfg.COAllNamespaces {
		return "all-namespaces"
	}
	return cfg.CONamespace
}
