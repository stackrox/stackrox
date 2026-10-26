package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func sampleDiff(files []string) changedFiles {
	return changedFiles{Base: strings.Repeat("a", 40), Head: strings.Repeat("b", 40), Files: files}
}

func TestClassify(t *testing.T) {
	groups := map[string]pathGroup{
		"docs": {Include: []string{`^(README\.md|CHANGELOG\.md)$`}},
		"go":   {Include: []string{`\.go$`}, Exclude: []string{`^ci/`}},
	}
	tests := map[string]struct {
		files               []string
		ignore              []string
		any, every, ignored []string
	}{
		"should classify both approved docs":                               {files: []string{"README.md", "CHANGELOG.md"}, any: []string{"docs"}, every: []string{"docs"}},
		"should retain tests for mixed changes":                            {files: []string{"README.md", "central/main.go"}, any: []string{"docs", "go"}},
		"should not infer docs from an extension":                          {files: []string{"docs/help.md", "testdata/expected.txt"}},
		"should not match every group for an empty diff":                   {files: []string{}},
		"should keep exclusions in the diff":                               {files: []string{"ci/test.go", "central/main.go"}, any: []string{"go"}},
		"should filter explicitly opted-in demo paths":                     {files: []string{"README.md", "ci/selection/main.go"}, ignore: []string{`^ci/selection/`}, any: []string{"docs"}, every: []string{"docs"}, ignored: []string{"ci/selection/main.go"}},
		"should not authorize skip when demo filtering removes everything": {files: []string{"ci/selection/main.go"}, ignore: []string{`^ci/selection/`}, ignored: []string{"ci/selection/main.go"}},
		"should retain product changes alongside ignored machinery":        {files: []string{"README.md", "ci/selection/main.go", "central/main.go"}, ignore: []string{`^ci/selection/`}, any: []string{"docs", "go"}, ignored: []string{"ci/selection/main.go"}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			facts, err := classify(sampleDiff(test.files), groups, test.ignore)
			require.NoError(t, err)
			require.True(t, facts.Complete)
			require.ElementsMatch(t, test.any, facts.Any)
			require.ElementsMatch(t, test.every, facts.Every)
			require.ElementsMatch(t, test.ignored, facts.Ignored)
		})
	}
}

func TestInvalidClassification(t *testing.T) {
	groups := map[string]pathGroup{"docs": {Include: []string{`\.md$`}}}
	for name, diff := range map[string]changedFiles{
		"should reject missing paths":     sampleDiff(nil),
		"should reject empty path":        sampleDiff([]string{""}),
		"should reject newline":           sampleDiff([]string{"README.md\n"}),
		"should reject traversal":         sampleDiff([]string{"../README.md"}),
		"should reject unnormalized path": sampleDiff([]string{"docs/../README.md"}),
		"should reject missing revision":  {Head: strings.Repeat("a", 40), Files: []string{"README.md"}},
	} {
		t.Run(name, func(t *testing.T) { _, err := classify(diff, groups, []string{`.*`}); require.Error(t, err) })
	}
	for name, group := range map[string]map[string]pathGroup{
		"should reject no groups":         {},
		"should reject no includes":       {"docs": {}},
		"should reject invalid regex":     {"docs": {Include: []string{"["}}},
		"should reject empty regex":       {"docs": {Include: []string{""}}},
		"should reject invalid exclusion": {"docs": {Include: []string{".*"}, Exclude: []string{"["}}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := classify(sampleDiff([]string{"README.md"}), group, nil)
			require.Error(t, err)
		})
	}
	_, err := classify(sampleDiff([]string{"README.md"}), groups, []string{"["})
	require.Error(t, err)
}

func TestDecide(t *testing.T) {
	for name, test := range map[string]struct {
		job      string
		facts    fileFacts
		decision string
	}{
		"should skip an enrolled docs-only workload": {"gke-qa-e2e-tests", fileFacts{Version: 1, Complete: true, Every: []string{"docs"}}, "skip"},
		"should run unknown targets":                 {"test-binary-build-commands", fileFacts{Version: 1, Complete: true, Every: []string{"docs"}}, "run"},
		"should run when data is unavailable":        {"gke-qa-e2e-tests", fileFacts{Version: 1, Every: []string{"docs"}}, "run"},
		"should run unsupported versions":            {"gke-qa-e2e-tests", fileFacts{Version: 2, Complete: true, Every: []string{"docs"}}, "run"},
		"should run mixed workloads":                 {"gke-qa-e2e-tests", fileFacts{Version: 1, Complete: true}, "run"},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.decision, decide(test.job, test.facts, []string{"gke-qa-e2e-tests"}).Decision)
		})
	}
}

func TestCommand(t *testing.T) {
	config := filepath.Join(t.TempDir(), "groups.json")
	require.NoError(t, os.WriteFile(config, []byte(`{"docs":{"include":["^README\\.md$"]}}`), 0o600))
	payload, err := json.Marshal(sampleDiff([]string{"README.md"}))
	require.NoError(t, err)
	var output bytes.Buffer
	require.NoError(t, run([]string{"--groups", config}, bytes.NewReader(payload), &output))
	var facts fileFacts
	require.NoError(t, json.Unmarshal(output.Bytes(), &facts))
	require.Equal(t, []string{"docs"}, facts.Every)
	for name, payload := range map[string]string{
		"should reject extra input":     string(payload) + ` {}`,
		"should reject unknown fields":  `{"base":"` + strings.Repeat("a", 40) + `","head":"` + strings.Repeat("b", 40) + `","files":[],"unexpected":true}`,
		"should reject malformed input": `{`,
	} {
		t.Run(name, func(t *testing.T) {
			var out bytes.Buffer
			require.Error(t, run([]string{"--groups", config}, strings.NewReader(payload), &out))
			require.Empty(t, out.String())
		})
	}
	require.Error(t, run([]string{"--groups", config, "extra"}, bytes.NewReader(payload), &output))
}

func TestGitChanges(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	git := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null")
		data, err := cmd.CombinedOutput()
		require.NoError(t, err, string(data))
		return strings.TrimSpace(string(data))
	}
	git("init", "--initial-branch=master")
	git("config", "user.email", "test@example.invalid")
	git("config", "user.name", "Selection Test")
	require.NoError(t, os.WriteFile("product.go", []byte("package product\n"), 0o600))
	git("add", "product.go")
	git("commit", "-m", "Initial product")
	base := git("rev-parse", "HEAD")
	git("switch", "-c", "piotr/test-head")
	git("mv", "product.go", "README.md")
	git("commit", "-m", "Move product path")
	head := git("rev-parse", "HEAD")
	diff, err := gitChanges(base, head)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"product.go", "README.md"}, diff.Files)
	git("switch", "master")
	require.NoError(t, os.WriteFile("base-only.go", []byte("package base\n"), 0o600))
	git("add", "base-only.go")
	git("commit", "-m", "Advance base")
	advanced := git("rev-parse", "HEAD")
	diff, err = gitChanges(advanced, head)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"product.go", "README.md"}, diff.Files, "base-only edits are not PR changes")
	_, err = gitChanges(strings.Repeat("f", 40), head)
	require.Error(t, err)
	_, err = gitChanges("--help", head)
	require.Error(t, err)
}
