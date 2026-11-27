package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_renderOutput(t *testing.T) {
	matrix := JobMatrix{
		Tests: []JobMatrixEntry{{Name: "test with <script> content"}},
	}

	jsonOutput, err := renderOutput(matrix, "json", "")
	require.NoError(t, err)
	assert.JSONEq(t, `{"tests":[{"name":"test with <script> content","source":"","source_ref":"","cluster_type":"","triggers":null,"description":""}],"duplicate_families":null}`, string(jsonOutput))

	htmlOutput, err := renderOutput(matrix, "html", "job-matrix.html.tmpl")
	require.NoError(t, err)
	assert.Contains(t, string(htmlOutput), "const data = {")
	assert.Contains(t, string(htmlOutput), `\u003cscript\u003e`)
	assert.NotContains(t, string(htmlOutput), "fetch(")

	_, err = renderOutput(matrix, "xml", "job-matrix.html.tmpl")
	assert.ErrorContains(t, err, `invalid output format "xml"`)
}

func Test_writeOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "output")
	require.NoError(t, writeOutput(path, []byte("output")))
	contents, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "output", string(contents))
}

type fakeProwSource struct {
	tests []ProwTest
	err   error
}

func (s fakeProwSource) Tests() ([]ProwTest, error) {
	return s.tests, s.err
}

func Test_buildJobMatrix(t *testing.T) {
	prow := fakeProwSource{
		tests: []ProwTest{
			{
				Name:              "gke-qa-e2e-tests",
				Branches:          []string{"master"},
				ClusterType:       "GKE (Google Kubernetes Engine)",
				Triggers:          []Trigger{TriggerPreMerge},
				RunsAutomatically: true,
				Description:       "Runs the main QA e2e suite.",
				TestFamily:        "qa-e2e-tests",
			},
			{
				Name:              "some-standalone-test",
				Branches:          []string{"master"},
				ClusterType:       "Unknown",
				Triggers:          []Trigger{TriggerPreMerge},
				RunsAutomatically: false,
				Optional:          true,
				Description:       "Unknown",
				TestFamily:        "some-standalone-test",
			},
		},
	}

	dir := t.TempDir()
	workflow := "name: QA e2e (OCP FIPS)\non:\n  workflow_call: {}\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "e2e-qa-tests-ocp-fips-konflux.yaml"), []byte(workflow), 0o600))
	backport := "name: Backport PR to release branch\non:\n  pull_request:\n    types: [closed, labeled]\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "backport-pr.yaml"), []byte(backport), 0o600))

	matrix, err := buildJobMatrix(prow, dir)
	require.NoError(t, err)
	require.Len(t, matrix.Tests, 4)

	// Only families with 2+ members should show up as duplicates, and a
	// family should be able to span both Prow and GitHub Actions entries.
	assert.Equal(t, map[string][]string{
		"qa-e2e-tests": {"gke-qa-e2e-tests", "e2e-qa-tests-ocp-fips-konflux.yaml"},
	}, matrix.DuplicateFamilies)

	byName := map[string]JobMatrixEntry{}
	for _, e := range matrix.Tests {
		byName[e.Name] = e
	}

	// RunsAutomatically/Optional are Prow-specific: populated (even when
	// false) for Prow entries, left nil for GitHub Actions ones.
	require.NotNil(t, byName["gke-qa-e2e-tests"].RunsAutomatically)
	assert.True(t, *byName["gke-qa-e2e-tests"].RunsAutomatically)
	assert.False(t, byName["gke-qa-e2e-tests"].Optional)

	require.NotNil(t, byName["some-standalone-test"].RunsAutomatically)
	assert.False(t, *byName["some-standalone-test"].RunsAutomatically)
	assert.True(t, byName["some-standalone-test"].Optional)

	assert.Nil(t, byName["e2e-qa-tests-ocp-fips-konflux.yaml"].RunsAutomatically)

	// RequiredLabel is GitHub-Actions-specific and should flow through from
	// the hand-curated metadata, not just the parsed trigger.
	assert.Empty(t, byName["gke-qa-e2e-tests"].RequiredLabel)
	assert.NotEmpty(t, byName["backport-pr.yaml"].RequiredLabel)
}

func Test_resolveWorkflowsDir(t *testing.T) {
	for name, tc := range map[string]struct {
		workflowsDirSet bool
		repoSet         bool
		branchSet       bool
		wantErr         string
		wantFetchCalled bool
	}{
		"workflows-dir only": {
			workflowsDirSet: true,
		},
		"neither set falls back to fetch defaults": {
			wantFetchCalled: true,
		},
		"repo set only": {
			repoSet:         true,
			wantFetchCalled: true,
		},
		"branch set only": {
			branchSet:       true,
			wantFetchCalled: true,
		},
		"workflows-dir and repo both set": {
			workflowsDirSet: true,
			repoSet:         true,
			wantErr:         "mutually exclusive",
		},
		"workflows-dir and branch both set": {
			workflowsDirSet: true,
			branchSet:       true,
			wantErr:         "mutually exclusive",
		},
		"workflows-dir, repo, and branch all set": {
			workflowsDirSet: true,
			repoSet:         true,
			branchSet:       true,
			wantErr:         "mutually exclusive",
		},
	} {
		t.Run(name, func(t *testing.T) {
			var fetchCalled bool
			var gotRepo, gotBranch string
			fetch := func(repo, branch string) (string, func(), error) {
				fetchCalled = true
				gotRepo, gotBranch = repo, branch
				return "fetched-dir", func() {}, nil
			}

			dir, cleanup, err := resolveWorkflowsDir("local-dir", tc.workflowsDirSet, "some/repo", "some-branch", tc.repoSet, tc.branchSet, fetch)

			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				assert.False(t, fetchCalled)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, cleanup)
			assert.Equal(t, tc.wantFetchCalled, fetchCalled)
			if tc.wantFetchCalled {
				assert.Equal(t, "fetched-dir", dir)
				assert.Equal(t, "some/repo", gotRepo)
				assert.Equal(t, "some-branch", gotBranch)
			} else {
				assert.Equal(t, "local-dir", dir)
			}
		})
	}
}

func Test_duplicateFamilies(t *testing.T) {
	entries := []JobMatrixEntry{
		{Name: "a", TestFamily: "shared"},
		{Name: "b", TestFamily: "shared"},
		{Name: "c", TestFamily: "lonely"},
		{Name: "d", TestFamily: ""},
	}

	assert.Equal(t, map[string][]string{
		"shared": {"a", "b"},
	}, duplicateFamilies(entries))
}
