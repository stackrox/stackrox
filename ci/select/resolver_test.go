package main

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	// TOML's typed decoder reports the offending key for type errors.
	typeErrors := map[string]string{
		"jobs-not-a-list.toml":    "jobs",
		"rule-not-a-table.toml":   "rules",
		"label-not-a-string.toml": "rules.label",
		"run-not-a-list.toml":     "rules.run",
		"paths-not-strings.toml":  "rules.paths",
	}
	if key, found := typeErrors[filepath.Base(path)]; found {
		assert.Contains(t, err.Error(), "read TOML rules")
		assert.Contains(t, err.Error(), `last key "`+key+`"`)
		return
	}
	expected = strings.ReplaceAll(expected, "'", `"`)
	expected = strings.TrimSuffix(expected, " in run")
	expected = strings.ReplaceAll(expected, " odd run mixes", " odd mixes")
	assert.Contains(t, err.Error(), expected)
}

func TestCurrentRules(t *testing.T) {
	m, err := loadMapping("../test-domains.toml")
	require.NoError(t, err)
	text, err := os.ReadFile("../decision-defaults")
	require.NoError(t, err)
	d, err := parseDefaults(string(text))
	require.NoError(t, err)
	cases := map[string]struct {
		files, labels, runs, skips []string
		complete                   bool
	}{
		"docs":                   {files: []string{"README.md"}, runs: []string{"style-check", "wait-for-images", "should-dispatch"}, complete: true},
		"shared product":         {files: []string{"sensor/common/foo.go", "central/policy/service.go"}, runs: []string{"go", "go-postgres", "sensor-integration-tests"}},
		"module":                 {files: []string{"go.mod"}, runs: m.Jobs, complete: true},
		"run all overrides docs": {files: []string{"README.md"}, labels: []string{"ci-run-all-tests"}, runs: m.Jobs, complete: true},
		"CI":                     {files: []string{"ci/select/gha.go"}, skips: []string{"e2e-byodb-tests", "go-postgres"}},
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

func TestMissingRulesUseDefaults(t *testing.T) {
	resolved, _, err := openDecision("testdata/missing.toml", "testdata/defaults", nil, nil, false, metadata{})
	require.NoError(t, err)
	assert.True(t, resolved.jobs["go"])
	assert.Contains(t, resolved.log, "using decision-defaults")
}

func TestResolverDesignExample(t *testing.T) {
	resolved, _, err := openDecision("testdata/rules.toml", "testdata/defaults",
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
	m, err := loadMapping("testdata/rules.toml")
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

func TestResolverUnmatchedFileUsesDefaults(t *testing.T) {
	m, err := loadMapping("../test-domains.toml")
	require.NoError(t, err)
	text, err := os.ReadFile("../decision-defaults")
	require.NoError(t, err)
	d, err := parseDefaults(string(text))
	require.NoError(t, err)
	expected, err := parseDecision(d.decision())
	require.NoError(t, err)
	resolved, err := resolve([]string{"notes/todo.txt"}, nil, m, d, metadata{})
	require.NoError(t, err)
	assert.Equal(t, expected, resolved.jobs)
}

func TestResolverCheckRequiresJob(t *testing.T) {
	rules := fixturePath(t, `version = 1
jobs = ["go", "go-check"]
[[rules]]
name = "matrix"
when = "always"
run = ["go-check"]
[[rules]]
name = "remaining"
when = "remaining"
default = ["*"]
[requires]
go-check = ["go"]
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
			rules: "testdata/corners.toml", decision: "unit\n", reason: "nothing-changed",
		},
		"unknown diff": {
			rules: "testdata/corners.toml", unknown: true, decision: "# no job runs\n", reason: "changed files are unknown",
		},
		"invalid version": {
			rules: "testdata/invalid/bad-version.toml", files: []string{"README.md"},
			decision: "# no job runs\n", reason: "unsupported rules version: 2",
		},
		"missing rules": {
			rules: "testdata/missing.toml", decision: "# no job runs\n", reason: "resolver failed:",
		},
		"broken TOML": {
			rules: "testdata/invalid/broken.toml", decision: "# no job runs\n", reason: "resolver failed:",
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

func TestResolverDefaultsMismatch(t *testing.T) {
	m, err := loadMapping("testdata/corners.toml")
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
			resolved, _, err := openDecision("testdata/corners.toml", defaultsPath, nil, nil, false, metadata{})
			require.NoError(t, err)
			assert.Equal(t, d.decision(), resolved.decision)
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
			_, _, err := openDecision("testdata/corners.toml", path, nil, nil, false, metadata{})
			require.Error(t, err)
		})
	}
}
