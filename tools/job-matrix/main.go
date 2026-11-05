// Command job-matrix inventories every test that runs in CI for
// stackrox/stackrox -- across both GitHub Actions and Prow -- into a single
// JSON or standalone HTML document, so redundant or near-duplicate test coverage (the same suite
// run against several cluster providers or versions) can be spotted.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
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
	workflowsDir := flag.String("workflows-dir", ".github/workflows", "directory containing GitHub Actions workflow files (mutually exclusive with -repo and -branch)")
	repo := flag.String("repo", "stackrox/stackrox", "owner/repo on GitHub to fetch GitHub Actions workflows from (mutually exclusive with -workflows-dir)")
	branch := flag.String("branch", "master", "branch to fetch GitHub Actions workflows from (mutually exclusive with -workflows-dir)")
	var output string
	flag.StringVar(&output, "o", "json", "output format: json or html")
	flag.StringVar(&output, "output", "json", "output format: json or html")
	outPath := flag.String("out", "", "optional path to write the output instead of stdout")
	flag.Parse()

	var workflowsDirSet, repoSet, branchSet bool
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "workflows-dir":
			workflowsDirSet = true
		case "repo":
			repoSet = true
		case "branch":
			branchSet = true
		}
	})

	resolvedWorkflowsDir, cleanup, err := resolveWorkflowsDir(*workflowsDir, workflowsDirSet, *repo, *branch, repoSet, branchSet, fetchWorkflowsDir)
	if err != nil {
		log.Fatal(err)
	}
	defer cleanup()

	matrix, err := buildJobMatrix(NewHTMLFileProwSource(*prowHTMLPath), resolvedWorkflowsDir)
	if err != nil {
		log.Fatal(err)
	}

	data, err := renderOutput(matrix, output, "tools/job-matrix/job-matrix.html.tmpl")
	if err != nil {
		log.Fatal(err)
	}

	if err := writeOutput(*outPath, data); err != nil {
		log.Fatal(err)
	}
}

func renderOutput(matrix JobMatrix, output, templatePath string) ([]byte, error) {
	jsonData, err := json.MarshalIndent(matrix, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling job matrix: %w", err)
	}
	jsonData = append(jsonData, '\n')
	if output == "json" {
		return jsonData, nil
	}
	if output != "html" {
		return nil, fmt.Errorf("invalid output format %q: must be json or html", output)
	}

	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return nil, fmt.Errorf("parsing HTML template %s: %w", templatePath, err)
	}
	var renderedOutput bytes.Buffer
	if err := tmpl.ExecuteTemplate(&renderedOutput, filepath.Base(templatePath), template.JS(jsonData)); err != nil {
		return nil, fmt.Errorf("rendering HTML template %s: %w", templatePath, err)
	}
	return renderedOutput.Bytes(), nil
}

func writeOutput(outPath string, data []byte) error {
	if outPath == "" {
		_, err := os.Stdout.Write(data)
		return err
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outPath, err)
	}
	return nil
}

// fetchFunc fetches GitHub Actions workflow definitions for repo at branch
// and returns their directory on disk plus a cleanup func.
type fetchFunc func(repo, branch string) (dir string, cleanup func(), err error)

// resolveWorkflowsDir decides whether to use workflowsDir as-is or fetch
// workflows from repo/branch, depending on which flags were explicitly set
// on the command line. workflowsDir and repo/branch are mutually exclusive:
// if neither is set, it falls back to fetching repo/branch's defaults.
func resolveWorkflowsDir(workflowsDir string, workflowsDirSet bool, repo, branch string, repoSet, branchSet bool, fetch fetchFunc) (string, func(), error) {
	if workflowsDirSet && (repoSet || branchSet) {
		return "", nil, fmt.Errorf("-workflows-dir is mutually exclusive with -repo and -branch")
	}
	if workflowsDirSet {
		return workflowsDir, func() {}, nil
	}
	return fetch(repo, branch)
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
