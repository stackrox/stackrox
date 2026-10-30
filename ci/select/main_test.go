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
	for name, args := range map[string][]string{
		"should validate local configuration": append([]string{"validate"}, base...),
		"should resolve local changed paths":  append(append([]string{"resolve"}, base...), "--files-file", files),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var output, diagnostic bytes.Buffer
			require.NoError(t, run(t.Context(), args, &output, &diagnostic))
			if args[0] == "validate" {
				assert.Contains(t, output.String(), "Validation passed")
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
			err := run(t.Context(), args, &output, &diagnostic)
			require.Error(t, err)
			assert.Empty(t, output.String())
			assert.Empty(t, diagnostic.String(), "main prints each error once")
			if name == "should require a command" {
				assert.ErrorContains(t, err, "a subcommand is required")
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
