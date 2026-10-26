package testing_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCIToolingSelection(t *testing.T) {
	f := newFixture(t)
	for name, test := range map[string]struct {
		paths, labels, selected []string
		all                     bool
	}{
		"detector edits select verification rather than ACS tests": {
			paths:    []string{".github/workflows/detect-changes.yml", "README.md"},
			selected: []string{"style-check", "selector-tests", "github-actions-lint"},
		},
		"unit workflow edits do not request every unit job": {
			paths:    []string{".github/workflows/unit-tests.yaml", "README.md"},
			selected: []string{"style-check", "selector-tests", "github-actions-lint"},
		},
		"classifier Go edits select its tests": {
			paths:    []string{"ci/native-selectors/file-facts/main.go", "README.md"},
			selected: []string{"style-check", "selector-tests"},
		},
		"reporter Go edits select its tests": {
			paths:    []string{"ci/native-selectors/selection-report/records.go", "README.md"},
			selected: []string{"style-check", "selector-tests"},
		},
		"reporter workflow edits select its contract tests": {
			paths:    []string{".github/workflows/ci-selection-report.yaml", "README.md"},
			selected: []string{"style-check", "selector-tests", "github-actions-lint"},
		},
		"tooling workflow edits select both tooling jobs": {
			paths:    []string{".github/workflows/native-ci-tests.yaml", "README.md"},
			selected: []string{"style-check", "selector-tests", "roxvet-tests", "github-actions-lint"},
		},
		"CI CRD fixture alone is not an ACS input": {
			paths:    []string{"ci/native-selectors/testing/testdata/example.crd.yaml"},
			selected: []string{"style-check", "selector-tests"},
		},
		"CI interface fixtures are not ACS inputs": {
			paths:    []string{"ci/native-selectors/testing/testdata/example.proto", "ci/native-selectors/testing/testdata/example.crd.yaml", "README.md"},
			selected: []string{"style-check", "selector-tests"},
		},
		"roxvet edits select its tests": {
			paths:    []string{"tools/roxvet/analyzers/validateimports/analyzer.go", "README.md"},
			selected: []string{"style-check", "roxvet-tests"},
		},
		"CI edits do not mask a shared ACS Go change": {
			paths:    []string{"ci/native-selectors/file-facts/main.go", "pkg/concurrency/context.go", "README.md"},
			selected: []string{"style-check", "selector-tests", "go", "go-postgres", "go-bench", "golangci-lint", "check-generated-files", "sensor-integration-tests"},
		},
		"other Go tools retain conservative product checks": {
			paths:    []string{"tools/generator/main.go", "README.md"},
			selected: []string{"style-check", "go", "golangci-lint", "check-generated-files"},
		},
		"shared dependencies retain E2E selection": {
			paths: []string{"ci/native-selectors/file-facts/main.go", "go.mod", "README.md"}, all: true,
		},
		"run-all remains an explicit coverage request": {
			paths: []string{"ci/native-selectors/file-facts/main.go", "README.md"}, labels: []string{"ci-run-all-tests"}, all: true,
		},
		"a named E2E request remains available on CI-only changes": {
			paths: []string{"ci/native-selectors/file-facts/main.go", "README.md"}, labels: []string{"e2e-qa-tests-gke"},
			selected: []string{"style-check", "selector-tests", "e2e-qa-tests-gke", "should-dispatch", "wait-for-images", "build-images"},
		},
		"the prototype PR selects only CI checks": {
			paths:    []string{".github/workflows/detect-changes.yml", ".github/workflows/unit-tests.yaml", ".github/workflows/e2e-dispatch.yaml", ".github/workflows/build.yaml", "ci/native-selectors/file-facts/main.go", "ci/native-selectors/testing/go.mod", "ci/native-selectors/testing/README.md", "tools/roxvet/analyzers/validateimports/analyzer.go"},
			selected: []string{"style-check", "selector-tests", "roxvet-tests", "github-actions-lint"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			labels := append(slices.Clone(test.labels), "ci-dispatcher-enforce")
			contexts := map[string]object{}
			context := func(workflow string) object {
				if ctx, ok := contexts[workflow]; ok {
					return ctx
				}
				ctx := f.context(t, test.paths, labels, options{workflow: workflow})
				contexts[workflow] = ctx
				return ctx
			}
			for target, selection := range f.local {
				want := test.all || slices.Contains(test.selected, target)
				require.Equal(t, want, f.candidate(t, target, context(selection.workflow)), target)
			}
			for _, target := range f.consumers {
				want := test.all || slices.Contains(test.selected, target)
				ctx := context("e2e-dispatch.yaml")
				require.Equal(t, want, f.candidate(t, target, ctx), target)
				if !want {
					require.False(t, f.execution(t, target, ctx), "unselected E2E work must not execute: %s", target)
				}
			}
		})
	}
}

func TestRoxvetTestScope(t *testing.T) {
	f := newFixture(t)
	var commands []string
	for _, step := range steps(obj(f.jobs("native-ci-tests.yaml")["roxvet-tests"])) {
		if step["id"] == "selection-record" {
			continue
		}
		if run := str(step["run"]); run != "" {
			commands = append(commands, run)
			require.Equal(t, "steps.selection.outputs.run != 'false'", step["if"])
		}
	}
	require.Equal(t, []string{"go test ./tools/roxvet/...", "go vet ./tools/roxvet/..."}, commands)
}
