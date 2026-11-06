package main

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stackrox/rox/ci/select/internal/engine/regexengine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	t.Parallel()
	files := writeTestFile(t, filepath.Join(t.TempDir(), "files"), "ui/app.tsx\nREADME.md\n")
	base := []string{"--rules", "../test-selection-rules.yaml"}
	for name, args := range map[string][]string{
		"should validate local configuration": append([]string{"validate"}, base...),
		"should resolve local changed paths":  append(append([]string{"resolve"}, base...), "--files-file", files),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var output, diagnostic bytes.Buffer
			require.NoError(t, run(t.Context(), args, &output, &diagnostic, regexengine.New()))
			if args[0] == "validate" {
				assert.Contains(t, output.String(), "Validation passed, with")
				assert.Contains(t, diagnostic.String(), "Warning:")
			} else {
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
				assert.Contains(t, diagnostic.String(), "frontend-backend-only")
			}
		})
	}
	// Validate failures are their own print channel: the command renders
	// the full lint report (errors and warnings) to the diagnostic stream
	// and returns a sentinel, so main stays quiet and nothing prints
	// twice. Every other failure is printed once by main.
	lintReported := map[string]bool{
		"should reject root-level rules": true,
		"should reject unreadable rules": true,
	}
	for name, args := range map[string][]string{
		"should require a command":               {},
		"should reject unknown commands":         {"prow"},
		"should reject GHA arguments":            {"gha", "go"},
		"should reject removed defaults flag":    {"validate", "--defaults", "missing"},
		"should reject unsupported flags":        {"validate", "--enforce"},
		"should reject missing flag values":      {"validate", "--rules"},
		"should reject rules on the GHA command": {"gha", "--rules", "missing"},
		"should reject root-level rules":         {"--rules", "missing", "validate"},
		"should reject positional arguments":     {"validate", "extra"},
		"should reject unreadable rules":         {"validate", "--rules", "missing"},
		"should require a diff file":             append([]string{"resolve"}, base...),
		"should reject unreadable diff file":     append(append([]string{"resolve"}, base...), "--files-file", "missing"),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var output, diagnostic bytes.Buffer
			err := run(t.Context(), args, &output, &diagnostic, regexengine.New())
			require.Error(t, err)
			assert.Empty(t, output.String())
			if lintReported[name] {
				assert.ErrorIs(t, err, errLintReported)
				assert.Contains(t, diagnostic.String(), "read rules:", name)
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
	const rules = `version: 1
jobs: [go]
rules:
  - name: docs
    when: any-file-matches
    paths: ['^docs/', '!^docs/internal/']
    skip: [go]
  - name: code
    when: any-file-matches
    paths: ['^docs/']
    run: [go]
  - name: remaining
    when: remaining
    run: ['*']
`
	path := writeTestFile(t, filepath.Join(t.TempDir(), "rules.yaml"), rules)
	var output, diagnostic bytes.Buffer
	err := run(t.Context(), []string{"resolve", "--rules", path}, &output, &diagnostic, regexengine.New())
	require.ErrorIs(t, err, errLintReported)
	assert.Empty(t, output.String())
	assert.Contains(t, diagnostic.String(), "001 Error:")
	assert.Contains(t, diagnostic.String(), "Warning:")
	assert.Contains(t, diagnostic.String(), "Result: 1 error, 1 warning.")
}

func TestValidatePassedLine(t *testing.T) {
	t.Parallel()
	clean := writeTestFile(t, filepath.Join(t.TempDir(), "rules.yaml"), `version: 1
jobs: [go]
rules:
  - name: docs
    when: any-file-matches
    paths: ['^docs/']
    skip: [go]
  - name: remaining
    when: remaining
    run: ['*']
`)
	var output, diagnostic bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"validate", "--rules", clean}, &output, &diagnostic, regexengine.New()))
	assert.Equal(t, "Validation passed\n", output.String())

	crossing := writeTestFile(t, filepath.Join(t.TempDir(), "rules.yaml"), `version: 1
jobs: [go]
rules:
  - name: special-input
    when: any-file-matches
    paths: ['^config/.*\.txt$']
    run: [go]
  - name: special-documentation
    when: any-file-matches
    paths: ['^config/special\.txt$']
    skip: [go]
  - name: remaining
    when: remaining
    run: ['*']
`)
	output.Reset()
	diagnostic.Reset()
	require.NoError(t, run(t.Context(), []string{"validate", "--rules", crossing}, &output, &diagnostic, regexengine.New()))
	assert.Equal(t, "Validation passed, with 1 warning\n", output.String())
	assert.Contains(t, diagnostic.String(), "Warning:")
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
			require.NoError(t, run(t.Context(), c.args, &output, &diagnostic, regexengine.New()))
			for _, text := range c.want {
				assert.Contains(t, output.String(), text)
			}
			assert.Empty(t, diagnostic.String())
		})
	}
}
