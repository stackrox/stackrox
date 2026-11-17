package main

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	t.Parallel()
	files := writeTestFile(t, filepath.Join(t.TempDir(), "files"), "ui/app.tsx\nREADME.md\n")
	base := []string{"--rules", "../test-selection-rules.yaml"}

	t.Run("should validate the committed configuration", func(t *testing.T) {
		t.Parallel()
		var output, diagnostic bytes.Buffer
		require.NoError(t, run(t.Context(), append([]string{"validate"}, base...), &output, &diagnostic))
		assert.Equal(t, "Validation passed\n", output.String())
	})

	t.Run("should resolve local changed paths", func(t *testing.T) {
		t.Parallel()
		var output, diagnostic bytes.Buffer
		args := append(append([]string{"resolve"}, base...), "--files-file", files)
		require.NoError(t, run(t.Context(), args, &output, &diagnostic))
		for _, target := range []string{"go", "go-bench", "go-postgres", "sensor-integration-tests"} {
			assert.Contains(t, output.String(), target+" skip\n")
		}
		for _, target := range e2eTargets() {
			if target == "e2e-ui-tests-gke" || target == "e2e-ui-tests-gke-konflux" {
				assert.Contains(t, output.String(), target+" run\n")
			} else {
				assert.Contains(t, output.String(), target+" skip\n")
			}
		}
		assert.Contains(t, diagnostic.String(), "go-unit")
	})

	// Every failure below is printed once by main, except the two lint
	// reports, which the command renders in full to the diagnostic stream
	// and then returns a sentinel so main stays quiet.
	lintReported := map[string]bool{
		"should reject unreadable rules": true,
	}
	for name, args := range map[string][]string{
		"should require a command":               {},
		"should reject unknown commands":         {"prow"},
		"should reject GHA arguments":            {"gha", "go"},
		"should reject unsupported flags":        {"validate", "--enforce"},
		"should reject missing flag values":      {"validate", "--rules"},
		"should reject rules on the GHA command": {"gha", "--rules", "missing"},
		"should reject positional arguments":     {"validate", "extra"},
		"should reject unreadable rules":         {"validate", "--rules", "missing"},
		"should require a diff file":             append([]string{"resolve"}, base...),
		"should reject unreadable diff file":     append(append([]string{"resolve"}, base...), "--files-file", "missing"),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var output, diagnostic bytes.Buffer
			err := run(t.Context(), args, &output, &diagnostic)
			require.Error(t, err)
			assert.Empty(t, output.String())
			if lintReported[name] {
				assert.ErrorIs(t, err, errLintReported)
				assert.Contains(t, diagnostic.String(), "read rules:")
			} else {
				assert.Empty(t, diagnostic.String(), "main prints each error once")
			}
			if name == "should require a command" {
				assert.ErrorContains(t, err, "a subcommand is required")
			}
		})
	}
}

func TestResolvePrintsTheFullLintReport(t *testing.T) {
	t.Parallel()
	// "go" is decided twice, which is a structural error.
	const rules = `version: 1
jobs: [go]
rules:
  - name: docs
    jobs: [go]
    skip_if_only_changed: ['^docs/']
  - name: code
    jobs: [go]
    run_if_changed: ['^central/']
`
	path := writeTestFile(t, filepath.Join(t.TempDir(), "rules.yaml"), rules)
	var output, diagnostic bytes.Buffer
	err := run(t.Context(), []string{"resolve", "--rules", path}, &output, &diagnostic)
	require.ErrorIs(t, err, errLintReported)
	assert.Empty(t, output.String())
	assert.Contains(t, diagnostic.String(), "001 Error:")
	assert.Contains(t, diagnostic.String(), `job "go" is decided more than once`)
}

func TestValidatePassedLine(t *testing.T) {
	t.Parallel()
	clean := writeTestFile(t, filepath.Join(t.TempDir(), "rules.yaml"), `version: 1
jobs: [go]
rules:
  - name: docs
    jobs: [go]
    skip_if_only_changed: ['^docs/']
`)
	var output, diagnostic bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"validate", "--rules", clean}, &output, &diagnostic))
	assert.Equal(t, "Validation passed\n", output.String())
	assert.Empty(t, diagnostic.String())
}

func TestRunHelp(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		args []string
		want []string
	}{
		"root":     {args: []string{"--help"}, want: []string{"validate", "resolve", "gha"}},
		"validate": {args: []string{"validate", "--help"}, want: []string{"--rules"}},
		"resolve":  {args: []string{"resolve", "--help"}, want: []string{"--rules", "--files-file"}},
		"gha":      {args: []string{"gha", "--help"}, want: []string{"gha"}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var output, diagnostic bytes.Buffer
			require.NoError(t, run(t.Context(), c.args, &output, &diagnostic))
			for _, text := range c.want {
				assert.Contains(t, output.String(), text)
			}
			assert.Empty(t, diagnostic.String())
		})
	}
}
