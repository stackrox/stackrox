package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type resolverCase struct {
	Name  string `json:"name"`
	Input struct {
		Files    []string `json:"files"`
		Labels   []string `json:"labels"`
		Rules    string   `json:"rules"`
		Defaults string   `json:"defaults"`
	} `json:"input"`
	Expect struct {
		Decision string `json:"decision"`
		Log      string `json:"log"`
		Error    string `json:"error"`
	} `json:"expect"`
}

func fixturePath(t *testing.T, value string) string {
	t.Helper()
	if strings.Contains(value, "\n") {
		path := filepath.Join(t.TempDir(), "config")
		require.NoError(t, os.WriteFile(path, []byte(value), 0o600))
		return path
	}
	return strings.TrimPrefix(value, "ci/select/")
}

func TestResolverCases(t *testing.T) {
	text, err := os.ReadFile("testdata/edge_cases.json")
	require.NoError(t, err)
	cases := []resolverCase{}
	require.NoError(t, json.Unmarshal(text, &cases))
	require.Len(t, cases, 18)
	byName := map[string]resolverCase{}
	for _, test := range cases {
		require.NotContains(t, byName, test.Name, "fixture names must be unique")
		byName[test.Name] = test
	}
	for name, test := range byName {
		t.Run(name, func(t *testing.T) {
			rules := fixturePath(t, test.Input.Rules)
			if test.Expect.Error != "" {
				assertRejectedRule(t, rules, test.Expect.Error)
				return
			}
			resolved, _, err := openDecision(rules, fixturePath(t, test.Input.Defaults),
				test.Input.Files, test.Input.Labels, test.Input.Files == nil,
				metadata{repo: "stackrox/stackrox", pr: "23035", commit: "a1b2c3d"})
			require.NoError(t, err)
			assert.Equal(t, test.Expect.Decision, resolved.decision)
			jobs, err := parseDecision(test.Expect.Decision)
			require.NoError(t, err)
			assert.Equal(t, jobs, resolved.jobs)
			lines := strings.Split(resolved.log, "\n")
			if strings.HasPrefix(lines[0], "# ") && len(lines) > 1 {
				lines[1] = "# <when>"
			}
			assert.Equal(t, test.Expect.Log, strings.Join(lines, "\n"))
		})
	}
}

func TestRejectedRules(t *testing.T) {
	text, err := os.ReadFile("testdata/rejected_rules.json")
	require.NoError(t, err)
	cases := []resolverCase{}
	require.NoError(t, json.Unmarshal(text, &cases))
	require.Len(t, cases, 33)
	byName := map[string]resolverCase{}
	for _, test := range cases {
		require.NotContains(t, byName, test.Name, "fixture names must be unique")
		byName[test.Name] = test
	}
	for name, test := range byName {
		t.Run(name, func(t *testing.T) {
			assertRejectedRule(t, fixturePath(t, test.Input.Rules), test.Expect.Error)
		})
	}
}

func assertRejectedRule(t *testing.T, path, expected string) {
	t.Helper()
	_, err := loadMapping(path)
	require.Error(t, err)
	typeErrors := map[string]string{
		"jobs-not-a-list.yaml":      "jobs",
		"rule-not-a-table.yaml":     "rules",
		"label-not-a-string.yaml":   "rules.label",
		"run-not-a-list.yaml":       "rules.run",
		"paths-not-strings.yaml":    "rules.paths",
		"requires-not-a-table.yaml": "requires",
	}
	if key, found := typeErrors[filepath.Base(path)]; found {
		assert.Contains(t, err.Error(), "read YAML rules")
		var typeError *json.UnmarshalTypeError
		require.ErrorAs(t, err, &typeError)
		assert.Equal(t, key, typeError.Field)
		return
	}
	expected = strings.ReplaceAll(expected, "'", `"`)
	expected = strings.TrimSuffix(expected, " in run")
	expected = strings.ReplaceAll(expected, " odd run mixes", " odd mixes")
	assert.Contains(t, err.Error(), expected)
}

func TestCurrentRules(t *testing.T) {
	m, err := loadMapping("../test-domains.yaml")
	require.NoError(t, err)
	text, err := os.ReadFile("../decision-defaults")
	require.NoError(t, err)
	d, err := parseDefaults(string(text))
	require.NoError(t, err)
	cases := map[string]struct {
		files, labels, runs, skips []string
		complete                   bool
	}{
		"docs":                   {files: []string{"README.md"}, runs: []string{"style-check", "build-images"}, complete: true},
		"UI":                     {files: []string{"ui/example.tsx"}, runs: []string{"style-check", "build-images", "ui", "ui-component", "gke-ui-e2e-tests"}, complete: true},
		"empty diff":             {runs: []string{"style-check", "build-images"}, complete: true},
		"shared product":         {files: []string{"sensor/common/foo.go", "central/policy/service.go"}, runs: []string{"go", "go-postgres", "sensor-integration-tests"}},
		"module":                 {files: []string{"go.mod"}, runs: m.Jobs, complete: true},
		"run all overrides docs": {files: []string{"README.md"}, labels: []string{"ci-run-all-tests"}, runs: m.Jobs, complete: true},
		"CI":                     {files: []string{"ci/select/gha.go"}, skips: []string{"e2e-byodb-tests", "go-postgres"}},
		"collector version":      {files: []string{"COLLECTOR_VERSION"}, runs: []string{"check-dependent-images-exist"}},
		"fact version":           {files: []string{"FACT_VERSION"}, runs: []string{"check-dependent-images-exist"}},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			resolved, err := resolve(test.files, test.labels, m, d, metadata{})
			require.NoError(t, err)
			if test.complete {
				expected := map[string]bool{}
				for _, job := range test.runs {
					expected[job] = true
				}
				assert.Equal(t, expected, resolved.jobs)
			}
			for _, job := range test.runs {
				assert.True(t, resolved.jobs[job], job)
			}
			for _, job := range test.skips {
				assert.False(t, resolved.jobs[job], job)
			}
		})
	}
}

func TestCIToolingSelection(t *testing.T) {
	text, err := os.ReadFile("testdata/ci-tooling-files")
	require.NoError(t, err)
	files := readList(string(text))
	require.NotEmpty(t, files)
	for name, test := range map[string]struct {
		files, labels []string
		enforce       bool
		want          string
		matchesCI     bool
	}{
		"enforced":          {files: files, enforce: true, want: "skip", matchesCI: true},
		"shadow":            {files: files, want: "run", matchesCI: true},
		"run all":           {files: files, labels: []string{"ci-run-all-tests"}, enforce: true, want: "run", matchesCI: true},
		"mixed product":     {files: append(slices.Clone(files), "sensor/common/example.go"), enforce: true, want: "skip"},
		"other action":      {files: append(slices.Clone(files), ".github/actions/job-preamble/action.yaml"), enforce: true, want: "skip"},
		"other tool source": {files: []string{"tools/roxvet/analyzers/validateimports/other.go"}, enforce: true, want: "skip"},
	} {
		t.Run(name, func(t *testing.T) {
			resolved, d, err := openDecision("../test-domains.yaml", "../decision-defaults",
				test.files, test.labels, false, metadata{})
			require.NoError(t, err)
			assert.NotContains(t, resolved.log, "using decision-defaults")
			assert.Equal(t, test.matchesCI, strings.Contains(resolved.log, `rule "ci-tooling" matches every changed file`))
			for _, job := range []string{
				"go-postgres", "sensor-integration-tests", "db-integration-tests",
				"e2e-byodb-tests", "e2e-db-backup-restore-tests", "e2e-gke-upgrade-tests",
				"e2e-nongroovy-tests-gke", "e2e-nongroovy-tests-gke-konflux",
			} {
				assert.Equal(t, test.want, gateAction(job, resolved.jobs, d, test.enforce, false), job)
			}
			assert.True(t, resolved.jobs["go"])
			assert.True(t, resolved.jobs["style-check"])
			assert.Equal(t, name == "run all", resolved.jobs["wait-for-images"])
			assert.True(t, resolved.jobs["build-images"])
			assert.Equal(t, name == "run all" || name == "mixed product", resolved.jobs["go-bench"])
		})
	}
}

func TestSelectorTestsGate(t *testing.T) {
	for name, test := range map[string]struct {
		files   []string
		labels  []string
		unknown bool
		enforce bool
		want    string
	}{
		"selector":             {files: []string{"ci/select/resolver.go"}, enforce: true, want: "run"},
		"selector tests":       {files: []string{"ci/select/resolver_test.go"}, enforce: true, want: "run"},
		"test fixtures":        {files: []string{"ci/select/testdata/edge_cases.json"}, enforce: true, want: "run"},
		"rules":                {files: []string{"ci/test-domains.yaml"}, enforce: true, want: "run"},
		"defaults":             {files: []string{"ci/decision-defaults"}, enforce: true, want: "run"},
		"Prow dispatcher":      {files: []string{".openshift-ci/dispatch.sh"}, enforce: true, want: "skip"},
		"build action":         {files: []string{".github/actions/ci-select/action.yaml"}, enforce: true, want: "run"},
		"test workflow":        {files: []string{".github/workflows/ci-select-tests.yaml"}, enforce: true, want: "run"},
		"GHA dispatcher":       {files: []string{".github/workflows/e2e-dispatch.yaml"}, enforce: true, want: "run"},
		"decision workflow":    {files: []string{".github/workflows/ci-decision.yaml"}, enforce: true, want: "run"},
		"CRD workflow":         {files: []string{".github/workflows/check-crd-compatibility.yaml"}, enforce: true, want: "run"},
		"scanner workflow":     {files: []string{".github/workflows/scanner-db-integration-tests.yaml"}, enforce: true, want: "run"},
		"style workflow":       {files: []string{".github/workflows/style.yaml"}, enforce: true, want: "run"},
		"build workflow":       {files: []string{".github/workflows/build.yaml"}, enforce: true, want: "run"},
		"unit workflow":        {files: []string{".github/workflows/unit-tests.yaml"}, enforce: true, want: "run"},
		"dependencies":         {files: []string{"go.mod"}, enforce: true, want: "run"},
		"dependency checksums": {files: []string{"go.sum"}, enforce: true, want: "run"},
		"mixed changes":        {files: []string{"ci/select/gha.go", "sensor/common/example.go"}, enforce: true, want: "run"},
		"product":              {files: []string{"sensor/common/example.go"}, enforce: true, want: "skip"},
		"unmatched":            {files: []string{"notes/todo.txt"}, enforce: true, want: "skip"},
		"UI":                   {files: []string{"ui/example.tsx"}, enforce: true, want: "skip"},
		"docs":                 {files: []string{"README.md"}, enforce: true, want: "skip"},
		"no changes":           {enforce: true, want: "skip"},
		"run all":              {files: []string{"README.md"}, labels: []string{"ci-run-all-tests"}, enforce: true, want: "run"},
		"shadow":               {files: []string{"README.md"}, want: "run"},
		"unknown diff":         {unknown: true, enforce: true, want: "run"},
	} {
		t.Run(name, func(t *testing.T) {
			resolved, d, err := openDecision("../test-domains.yaml", "../decision-defaults",
				test.files, test.labels, test.unknown, metadata{})
			require.NoError(t, err)
			assert.Equal(t, test.want, gateAction("selector-tests", resolved.jobs, d, test.enforce, false))
		})
	}
}

func TestMissingRulesUseDefaults(t *testing.T) {
	resolved, _, err := openDecision("testdata/missing.yaml", "testdata/defaults", nil, nil, false, metadata{})
	require.NoError(t, err)
	assert.True(t, resolved.jobs["go"])
	assert.Contains(t, resolved.log, "using decision-defaults")
}

func TestResolverDesignExample(t *testing.T) {
	resolved, _, err := openDecision("testdata/rules.yaml", "testdata/defaults",
		[]string{"README.md"}, nil, false,
		metadata{repo: "stackrox/stackrox", pr: "23035", commit: "a1b2c3d12345"})
	require.NoError(t, err)
	assert.Equal(t, "style-check\nwait-for-images\nshould-dispatch\n", resolved.decision)
	assert.Equal(t, map[string]bool{"style-check": true, "wait-for-images": true, "should-dispatch": true}, resolved.jobs)
	lines := strings.Split(resolved.log, "\n")
	assert.Equal(t, "# stackrox/stackrox PR 23035 a1b2c3d", lines[0])
	_, err = time.Parse(time.RFC3339, strings.TrimPrefix(lines[1], "# "))
	require.NoError(t, err)
	for _, expected := range []string{
		"001. rule \"docs-only\" matches every changed file\n     README.md\n",
		`002. rule "docs-only" skips target "go"`,
		`009. rule "docs-only" skips target "e2e-nongroovy-tests"`,
		`017. clash on target "style-check"; rule "style" says run, rule "docs-only" says skip; run wins`,
		`018. clash on target "wait-for-images"; rule "image-wait" says run, rule "docs-only" says skip; run wins`,
		`019. added target "should-dispatch" because target "wait-for-images" requires it`,
	} {
		assert.Contains(t, resolved.log, expected)
	}
	assert.NotContains(t, resolved.log, "because file")
}

func TestResolverMixedCIToolingAndProduct(t *testing.T) {
	m, err := loadMapping("testdata/rules.yaml")
	require.NoError(t, err)
	text, err := os.ReadFile("testdata/defaults")
	require.NoError(t, err)
	d, err := parseDefaults(string(text))
	require.NoError(t, err)
	resolved, err := resolve([]string{".github/workflows/style.yaml", "sensor/common/foo.go"}, nil, m, d, metadata{})
	require.NoError(t, err)
	assert.True(t, resolved.jobs["e2e-byodb-tests"])
	assert.True(t, resolved.jobs["go"])
	assert.Contains(t, resolved.log, `rule "style" runs target "style-check"`)
	assert.NotContains(t, resolved.log, `rule "ci-tooling" skips`)
}

func TestResolverUnmatchedFileSkipsRemaining(t *testing.T) {
	m, err := loadMapping("../test-domains.yaml")
	require.NoError(t, err)
	text, err := os.ReadFile("../decision-defaults")
	require.NoError(t, err)
	d, err := parseDefaults(string(text))
	require.NoError(t, err)
	resolved, err := resolve([]string{"notes/todo.txt"}, nil, m, d, metadata{})
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{"style-check": true, "build-images": true}, resolved.jobs)
	assert.NotContains(t, resolved.log, "takes default")
	assert.Contains(t, resolved.log, `rule "skip-selector-tests" skips target "selector-tests"`)
}

func TestResolverCheckRequiresJob(t *testing.T) {
	rules := fixturePath(t, `version: 1
jobs:
  - go
  - go-check

rules:
  - name: matrix
    when: always
    run:
      - go-check
  - name: remaining
    when: remaining
    default:
      - '*'
requires:
  go-check:
    - go
`)
	m, err := loadMapping(rules)
	require.NoError(t, err)
	d, err := parseDefaults("go skip\ngo-check skip\n")
	require.NoError(t, err)
	resolved, err := resolve(nil, nil, m, d, metadata{})
	require.NoError(t, err)
	assert.Equal(t, "go-check\ngo\n", resolved.decision)
	assert.Equal(t, map[string]bool{"go": true, "go-check": true}, resolved.jobs)
	assert.Contains(t, resolved.log, `002. added target "go" because target "go-check" requires it`)
	assert.NotContains(t, resolved.log, `takes default "skip" for target "go"`)
}

func TestOpenDecisionFallbacks(t *testing.T) {
	for name, test := range map[string]struct {
		rules            string
		files            []string
		unknown          bool
		decision, reason string
	}{
		"empty known diff": {
			rules: "testdata/corners.yaml", decision: "unit\n", reason: "nothing-changed",
		},
		"unknown diff": {
			rules: "testdata/corners.yaml", unknown: true, decision: "# no job runs\n", reason: "changed files are unknown",
		},
		"invalid version": {
			rules: "testdata/invalid/bad-version.yaml", files: []string{"README.md"},
			decision: "# no job runs\n", reason: "unsupported rules version: 2",
		},
		"missing rules": {
			rules: "testdata/missing.yaml", decision: "# no job runs\n", reason: "resolver failed:",
		},
		"broken YAML": {
			rules: "testdata/invalid/broken.yaml", decision: "# no job runs\n", reason: "resolver failed:",
		},
	} {
		t.Run(name, func(t *testing.T) {
			resolved, owned, err := openDecision(test.rules, "testdata/corners-defaults",
				test.files, nil, test.unknown, metadata{})
			require.NoError(t, err)
			assert.Equal(t, test.decision, resolved.decision)
			jobs, err := parseDecision(test.decision)
			require.NoError(t, err)
			assert.Equal(t, jobs, resolved.jobs)
			assert.Equal(t, "skip", owned.actions["unit"])
			assert.Contains(t, resolved.log, test.reason)
			if name != "empty known diff" {
				assert.Contains(t, resolved.log, "using decision-defaults")
			}
		})
	}
}

func TestFallbackRunLabels(t *testing.T) {
	for name, test := range map[string]struct {
		rules, defaults string
		labels, runs    []string
		unknown         bool
	}{
		"should retain dependencies for a forced job with an unknown diff": {
			rules: "testdata/corners.yaml", defaults: "testdata/corners-defaults", unknown: true,
			labels: []string{"suite"}, runs: []string{"suite", "images", "extra"},
		},
		"should run all jobs when rules are missing": {
			rules: "testdata/missing.yaml", defaults: "testdata/corners-defaults",
			labels: []string{"ci-run-all-tests"}, runs: []string{"unit", "lint", "suite", "images", "extra"},
		},
		"should honor a GHA job label when rules are invalid": {
			rules: "testdata/invalid/broken.yaml", defaults: "../decision-defaults",
			labels: []string{"e2e-qa-tests-gke"}, runs: []string{"e2e-qa-tests-gke", "wait-for-images", "build-images"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			resolved, _, err := openDecision(test.rules, test.defaults, nil, test.labels, test.unknown, metadata{})
			require.NoError(t, err)
			assert.Contains(t, resolved.log, "using decision-defaults")
			assert.Contains(t, resolved.log, "run labels override skipped defaults")
			for _, job := range test.runs {
				assert.True(t, resolved.jobs[job], job)
			}
			jobs, err := parseDecision(resolved.decision)
			require.NoError(t, err)
			assert.Equal(t, jobs, resolved.jobs)
		})
	}
}

func TestResolverDefaultsMismatch(t *testing.T) {
	m, err := loadMapping("testdata/corners.yaml")
	require.NoError(t, err)
	for name, text := range map[string]string{
		"missing target": "unit run\n",
		"extra target":   "unit skip\nlint skip\nsuite skip\nimages skip\nextra skip\nunowned run\n",
	} {
		t.Run(name, func(t *testing.T) {
			d, err := parseDefaults(text)
			require.NoError(t, err)
			_, err = resolve(nil, nil, m, d, metadata{})
			require.Error(t, err)
			if name == "missing target" {
				assert.Contains(t, err.Error(), "decision-defaults is missing lint")
			} else {
				assert.Contains(t, err.Error(), "decision-defaults has unknown target unowned")
			}
			defaultsPath := fixturePath(t, text)
			resolved, _, err := openDecision("testdata/corners.yaml", defaultsPath, nil, nil, false, metadata{})
			require.NoError(t, err)
			assert.Equal(t, d.decision(nil), resolved.decision)
			assert.Contains(t, resolved.log, "using decision-defaults")
		})
	}
}

func TestOpenDecisionUnusableDefaults(t *testing.T) {
	for name, path := range map[string]string{
		"missing": filepath.Join(t.TempDir(), "missing"),
		"invalid": fixturePath(t, "unit maybe\n"),
	} {
		t.Run(name, func(t *testing.T) {
			_, _, err := openDecision("testdata/corners.yaml", path, nil, nil, false, metadata{})
			require.Error(t, err)
		})
	}
}
