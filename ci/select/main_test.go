package main

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stackrox/rox/ci/select/internal/ruleengine/jobcentric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testRules is a small but representative rules file: one run_if_changed, one
// skip_if_only_changed, and one always_run, covering every job exactly once.
const testRules = `jobs: [go, postgres, lint]
rules:
  - name: go-unit
    jobs: [go]
    skip_if_only_changed: ['^ui/', '^.*\.md$']
  - name: db
    jobs: [postgres]
    run_if_changed: ['^central/', '^pkg/']
  - name: lint
    jobs: [lint]
    always_run: true
`

func TestPlan(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	rules := writeTestFile(t, filepath.Join(dir, "rules.yaml"), testRules)
	// A docs-only change: go is skipped (only docs), postgres is skipped (no
	// central/pkg match), lint always runs.
	files := writeTestFile(t, filepath.Join(dir, "changed"), "README.md\n")
	var output, diagnostic bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"plan", "--rules", rules, "--files-file", files}, &output, &diagnostic, jobcentric.New()))
	assert.Contains(t, output.String(), "-- plan --")
	assert.Contains(t, output.String(), "go skip\n")
	assert.Contains(t, output.String(), "postgres skip\n")
	assert.Contains(t, output.String(), "lint run\n")
	// The changed paths flow through to the engine and show in the trace.
	assert.Contains(t, diagnostic.String(), "README.md")
}

func TestPlanRunsOnMatch(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	rules := writeTestFile(t, filepath.Join(dir, "rules.yaml"), testRules)
	// A central/ change: go runs (not docs-only), postgres runs (central match).
	files := writeTestFile(t, filepath.Join(dir, "changed"), "central/sac/client.go\n")
	var output, diagnostic bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"plan", "--rules", rules, "--files-file", files}, &output, &diagnostic, jobcentric.New()))
	assert.Contains(t, output.String(), "go run\n")
	assert.Contains(t, output.String(), "postgres run\n")
	assert.Contains(t, output.String(), "lint run\n")
}

func TestPlanWithoutChangedPaths(t *testing.T) {
	t.Parallel()
	rules := writeTestFile(t, filepath.Join(t.TempDir(), "rules.yaml"), testRules)
	var output, diagnostic bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"plan", "--rules", rules}, &output, &diagnostic, jobcentric.New()))
	// An empty diff runs every job.
	assert.Contains(t, output.String(), "go run\n")
	assert.Contains(t, output.String(), "postgres run\n")
	assert.Contains(t, output.String(), "lint run\n")
}

func TestPlanWritesStepSummary(t *testing.T) {
	rules := writeTestFile(t, filepath.Join(t.TempDir(), "rules.yaml"), testRules)
	summary := filepath.Join(t.TempDir(), "summary")
	t.Setenv("GITHUB_STEP_SUMMARY", summary)
	var output, diagnostic bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"plan", "--rules", rules}, &output, &diagnostic, jobcentric.New()))
	assert.FileExists(t, summary)
	data := readFile(t, summary)
	assert.Contains(t, data, "## CI selection plan")
	assert.Contains(t, data, "go run")
}

func TestValidate(t *testing.T) {
	t.Parallel()
	rules := writeTestFile(t, filepath.Join(t.TempDir(), "rules.yaml"), testRules)
	var output, diagnostic bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"validate", "--rules", rules}, &output, &diagnostic, jobcentric.New()))
	assert.Contains(t, output.String(), "Validation passed")
}

func TestValidateReportsFindings(t *testing.T) {
	t.Parallel()
	// job "postgres" is in the inventory but no rule decides it.
	bad := writeTestFile(t, filepath.Join(t.TempDir(), "rules.yaml"),
		"jobs: [go, postgres]\nrules:\n  - name: go-unit\n    jobs: [go]\n    always_run: true\n")
	var output, diagnostic bytes.Buffer
	err := run(t.Context(), []string{"validate", "--rules", bad}, &output, &diagnostic, jobcentric.New())
	require.Error(t, err)
	// The report is on stderr; main suppresses a second print of the sentinel.
	assert.Contains(t, diagnostic.String(), "no rule decides it")
}

func TestRunErrors(t *testing.T) {
	t.Parallel()
	validRules := writeTestFile(t, filepath.Join(t.TempDir(), "ok.yaml"), testRules)
	malformed := writeTestFile(t, filepath.Join(t.TempDir(), "bad.yaml"), "jobs: [\n")
	for name, c := range map[string]struct {
		args    []string
		message string
	}{
		"should require a command":           {args: nil, message: "a subcommand is required"},
		"should reject unknown commands":     {args: []string{"resolve"}},
		"should reject positional arguments": {args: []string{"plan", "extra"}},
		"should reject unknown flags":        {args: []string{"plan", "--enforce"}},
		"should report unreadable rules":     {args: []string{"plan", "--rules", "missing"}, message: "read rules"},
		"should report unreadable files":     {args: []string{"plan", "--rules", validRules, "--files-file", "missing"}, message: "read changed paths"},
		"should report malformed rules":      {args: []string{"plan", "--rules", malformed}, message: "parse rules"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var output, diagnostic bytes.Buffer
			err := run(t.Context(), c.args, &output, &diagnostic, jobcentric.New())
			require.Error(t, err)
			if c.message != "" {
				assert.ErrorContains(t, err, c.message)
			}
		})
	}
}

func TestRunHelp(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		args []string
		want []string
	}{
		"root":     {args: []string{"--help"}, want: []string{"plan", "validate"}},
		"plan":     {args: []string{"plan", "--help"}, want: []string{"--rules", "--files-file"}},
		"validate": {args: []string{"validate", "--help"}, want: []string{"--rules"}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var output, diagnostic bytes.Buffer
			require.NoError(t, run(t.Context(), c.args, &output, &diagnostic, jobcentric.New()))
			for _, text := range c.want {
				assert.Contains(t, output.String(), text)
			}
		})
	}
}
