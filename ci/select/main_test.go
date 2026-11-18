package main

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stackrox/rox/ci/select/internal/ruleengine/alwaysrun"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlan(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	rules := writeTestFile(t, filepath.Join(dir, "rules.yaml"), "version: 1\njobs: [go, go-postgres]\n")
	files := writeTestFile(t, filepath.Join(dir, "changed"), "ui/app.tsx\nREADME.md\n")
	var output, diagnostic bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"plan", "--rules", rules, "--files-file", files}, &output, &diagnostic, alwaysrun.New()))
	assert.Contains(t, output.String(), "-- plan --")
	assert.Contains(t, output.String(), "go run\n")
	assert.Contains(t, output.String(), "go-postgres run\n")
	// The changed paths flow through to the engine and show in the trace.
	assert.Contains(t, diagnostic.String(), "2 changed path(s)")
}

func TestPlanWithoutChangedPaths(t *testing.T) {
	t.Parallel()
	rules := writeTestFile(t, filepath.Join(t.TempDir(), "rules.yaml"), "version: 1\njobs: [go]\n")
	var output, diagnostic bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"plan", "--rules", rules}, &output, &diagnostic, alwaysrun.New()))
	assert.Contains(t, output.String(), "go run\n")
	assert.Contains(t, diagnostic.String(), "0 changed path(s)")
}

func TestPlanWritesStepSummary(t *testing.T) {
	rules := writeTestFile(t, filepath.Join(t.TempDir(), "rules.yaml"), "version: 1\njobs: [go]\n")
	summary := filepath.Join(t.TempDir(), "summary")
	t.Setenv("GITHUB_STEP_SUMMARY", summary)
	var output, diagnostic bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"plan", "--rules", rules}, &output, &diagnostic, alwaysrun.New()))
	assert.FileExists(t, summary)
	data := readFile(t, summary)
	assert.Contains(t, data, "## CI selection plan")
	assert.Contains(t, data, "go run")
}

func TestRunErrors(t *testing.T) {
	t.Parallel()
	emptyJobs := writeTestFile(t, filepath.Join(t.TempDir(), "empty.yaml"), "version: 1\njobs: []\n")
	malformed := writeTestFile(t, filepath.Join(t.TempDir(), "bad.yaml"), "version: 1\njobs: [\n")
	for name, c := range map[string]struct {
		args    []string
		message string
	}{
		"should require a command":           {args: nil, message: "a subcommand is required"},
		"should reject unknown commands":     {args: []string{"resolve"}},
		"should reject positional arguments": {args: []string{"plan", "extra"}},
		"should reject unknown flags":        {args: []string{"plan", "--enforce"}},
		"should report unreadable rules":     {args: []string{"plan", "--rules", "missing"}, message: "read rules"},
		"should report unreadable files":     {args: []string{"plan", "--rules", emptyJobs, "--files-file", "missing"}, message: "read changed paths"},
		"should report rules with no jobs":   {args: []string{"plan", "--rules", emptyJobs}, message: "list no jobs"},
		"should report malformed rules":      {args: []string{"plan", "--rules", malformed}, message: "parse rules"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var output, diagnostic bytes.Buffer
			err := run(t.Context(), c.args, &output, &diagnostic, alwaysrun.New())
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
		"root": {args: []string{"--help"}, want: []string{"plan"}},
		"plan": {args: []string{"plan", "--help"}, want: []string{"--rules", "--files-file"}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var output, diagnostic bytes.Buffer
			require.NoError(t, run(t.Context(), c.args, &output, &diagnostic, alwaysrun.New()))
			for _, text := range c.want {
				assert.Contains(t, output.String(), text)
			}
		})
	}
}
