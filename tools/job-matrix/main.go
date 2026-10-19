// Command job-matrix inventories every test that runs in CI for
// stackrox/stackrox -- across both GitHub Actions and Prow -- into a single
// JSON file, so redundant or near-duplicate test coverage (the same suite
// run against several cluster providers or versions) can be spotted.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
)

// JobMatrixEntry is the common shape both Prow tests and GitHub Actions
// workflows are rendered into.
type JobMatrixEntry struct {
	Name        string    `json:"name"`
	Source      string    `json:"source"`
	SourceRef   string    `json:"source_ref"`
	Branches    []string  `json:"branches,omitempty"`
	ClusterType string    `json:"cluster_type"`
	Triggers    []Trigger `json:"triggers"`
	// RunsAutomatically is only populated for Prow entries (nil for GitHub
	// Actions, where the concept doesn't apply): false means a presubmit is
	// opt-in only (ci-operator always_run: false) and won't run on a PR
	// unless someone comments "/test <name>".
	RunsAutomatically *bool `json:"runs_automatically,omitempty"`
	Optional          bool  `json:"optional,omitempty"`
	// RequiredLabel is only populated for GitHub Actions entries: when set,
	// the workflow only does anything on a PR once this label is applied,
	// rather than running unconditionally.
	RequiredLabel string `json:"required_label,omitempty"`
	Description   string `json:"description"`
	TestFamily    string `json:"test_family,omitempty"`
}

// JobMatrix is the top-level shape written to the output file.
type JobMatrix struct {
	Tests             []JobMatrixEntry    `json:"tests"`
	DuplicateFamilies map[string][]string `json:"duplicate_families"`
}

func main() {
	prowHTMLPath := flag.String("prow-html", "tmp/prow-jobs.html", "path to the Prow ci-operator test-matrix HTML export")
	workflowsDir := flag.String("workflows-dir", ".github/workflows", "directory containing GitHub Actions workflow files")
	outPath := flag.String("out", "tmp/job-matrix.json", "path to write the resulting job matrix JSON to")
	flag.Parse()

	matrix, err := buildJobMatrix(NewHTMLFileProwSource(*prowHTMLPath), *workflowsDir)
	if err != nil {
		log.Fatal(err)
	}

	data, err := json.MarshalIndent(matrix, "", "  ")
	if err != nil {
		log.Fatalf("marshaling job matrix: %v", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(*outPath, data, 0o644); err != nil {
		log.Fatalf("writing %s: %v", *outPath, err)
	}

	fmt.Printf("wrote %d tests (%d duplicate families) to %s\n", len(matrix.Tests), len(matrix.DuplicateFamilies), *outPath)
}

func buildJobMatrix(prowSource ProwSource, workflowsDir string) (JobMatrix, error) {
	prowTests, err := prowSource.Tests()
	if err != nil {
		return JobMatrix{}, fmt.Errorf("reading Prow tests: %w", err)
	}

	ghaWorkflows, err := GHAWorkflows(workflowsDir)
	if err != nil {
		return JobMatrix{}, fmt.Errorf("reading GitHub Actions workflows: %w", err)
	}

	entries := make([]JobMatrixEntry, 0, len(prowTests)+len(ghaWorkflows))
	for _, t := range prowTests {
		runsAutomatically := t.RunsAutomatically
		entries = append(entries, JobMatrixEntry{
			Name:              t.Name,
			Source:            "Prow",
			SourceRef:         fmt.Sprintf("ci-operator test %q", t.Name),
			Branches:          t.Branches,
			ClusterType:       t.ClusterType,
			Triggers:          t.Triggers,
			RunsAutomatically: &runsAutomatically,
			Optional:          t.Optional,
			Description:       t.Description,
			TestFamily:        t.TestFamily,
		})
	}
	for _, w := range ghaWorkflows {
		entries = append(entries, JobMatrixEntry{
			Name:          w.FileName,
			Source:        "GitHub Actions",
			SourceRef:     ".github/workflows/" + w.FileName,
			ClusterType:   w.ClusterType,
			Triggers:      w.Triggers,
			RequiredLabel: w.RequiredLabel,
			Description:   w.Description,
			TestFamily:    w.TestFamily,
		})
	}

	return JobMatrix{
		Tests:             entries,
		DuplicateFamilies: duplicateFamilies(entries),
	}, nil
}

// duplicateFamilies groups entry names by TestFamily, keeping only families
// with two or more members -- i.e. the same underlying test showing up more
// than once, whether that's the same suite against different cluster
// providers, different platform versions, or both GitHub Actions and Prow.
func duplicateFamilies(entries []JobMatrixEntry) map[string][]string {
	families := map[string][]string{}
	for _, e := range entries {
		if e.TestFamily == "" {
			continue
		}
		families[e.TestFamily] = append(families[e.TestFamily], e.Name)
	}
	for family, names := range families {
		if len(names) < 2 {
			delete(families, family)
		}
	}
	return families
}
