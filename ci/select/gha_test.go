package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGHAOutputs(t *testing.T) {
	t.Setenv("CI_SELECT_ROOT", "../..")
	t.Setenv("CI_BASE", "missing-ref")
	t.Setenv("CI_HEAD", "HEAD")
	t.Setenv("CI_ENFORCE", "true")
	path := filepath.Join(t.TempDir(), "output")
	t.Setenv("GITHUB_OUTPUT", path)
	var output bytes.Buffer
	require.NoError(t, ghaCI(t.Context(), []string{"gke-qa-e2e-tests"}, &output, &bytes.Buffer{}))
	text, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "action=skip\n", string(text))
	assert.Equal(t, string(text), output.String())
}

func TestDispatchPlan(t *testing.T) {
	d := defaults{actions: map[string]string{"e2e-nongroovy-tests-gke": "run", "wait-for-images": "run"}}
	for name, test := range map[string]struct {
		input                                                    dispatchInput
		resolved                                                 *result
		nongroovy, nongroovyKonflux, byodb, waitGHA, waitKonflux bool
	}{
		"push":                       {input: dispatchInput{}, nongroovy: true, nongroovyKonflux: true, byodb: true, waitGHA: true, waitKonflux: true},
		"ready PR":                   {input: dispatchInput{pullRequest: true}, nongroovy: true, waitGHA: true},
		"draft":                      {input: dispatchInput{pullRequest: true, draft: true}},
		"UI":                         {input: dispatchInput{pullRequest: true, uiOnly: true}, waitGHA: true},
		"explicit label":             {input: dispatchInput{pullRequest: true, draft: true, labels: []string{"e2e-byodb-tests"}}, byodb: true, waitGHA: true},
		"Konflux":                    {input: dispatchInput{pullRequest: true, labels: []string{"konflux-build"}}, nongroovy: true, nongroovyKonflux: true, waitGHA: true, waitKonflux: true},
		"explicit nongroovy Konflux": {input: dispatchInput{pullRequest: true, draft: true, labels: []string{"e2e-nongroovy-tests-gke-konflux"}}, nongroovyKonflux: true, waitGHA: true},
		"enforce skip":               {input: dispatchInput{pullRequest: true, enforce: true}, resolved: &result{jobs: map[string]bool{}}},
		"shadow":                     {input: dispatchInput{pullRequest: true}, resolved: &result{jobs: map[string]bool{}}, nongroovy: true, waitGHA: true},
		"fork":                       {input: dispatchInput{pullRequest: true, fork: true, enforce: true}, resolved: &result{jobs: map[string]bool{"e2e-nongroovy-tests-gke": true, "wait-for-images": true}}},
	} {
		t.Run(name, func(t *testing.T) {
			plan := dispatchPlan(test.input, test.resolved, d)
			jobs := map[string]bool{}
			require.NoError(t, json.Unmarshal([]byte(plan["jobs"]), &jobs))
			assert.Equal(t, test.nongroovy, jobs["e2e-nongroovy-tests-gke"])
			assert.Equal(t, test.nongroovyKonflux, jobs["e2e-nongroovy-tests-gke-konflux"])
			assert.NotContains(t, jobs, "e2e-nongroovy-tests")
			assert.Equal(t, test.byodb, jobs["e2e-byodb-tests"])
			assert.Equal(t, fmt.Sprint(test.waitGHA), plan["wait-for-gha-images"])
			assert.Equal(t, fmt.Sprint(test.waitKonflux), plan["wait-for-konflux-images"])
		})
	}
}

func TestDispatchInventory(t *testing.T) {
	m, err := loadMapping("../test-domains.yaml")
	require.NoError(t, err)
	text, err := os.ReadFile("../decision-defaults")
	require.NoError(t, err)
	d, err := parseDefaults(string(text))
	require.NoError(t, err)
	for _, job := range dispatchJobs {
		assert.Contains(t, m.Jobs, job.name)
		assert.Contains(t, d.actions, job.name)
	}
	for name, test := range map[string]struct {
		files, labels []string
		konflux       bool
	}{
		"docs with Konflux": {files: []string{"README.md"}, labels: []string{"konflux-build"}},
		"CI with Konflux":   {files: []string{"ci/select/gha.go"}, labels: []string{"konflux-build"}},
		"run all without images": {
			files: []string{"README.md"}, labels: []string{"ci-run-all-tests"},
		},
		"run all with images": {
			files: []string{"README.md"}, labels: []string{"ci-run-all-tests", "konflux-build"}, konflux: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			resolved, err := resolve(test.files, test.labels, m, d, metadata{})
			require.NoError(t, err)
			input := dispatchInput{pullRequest: true, enforce: true, labels: test.labels}
			plan := dispatchPlan(input, &resolved, d)
			jobs := map[string]bool{}
			require.NoError(t, json.Unmarshal([]byte(plan["jobs"]), &jobs))
			for _, job := range dispatchJobs {
				if job.requireKonflux {
					assert.Equal(t, test.konflux, jobs[job.name], job.name)
				}
			}
		})
	}
}

func TestGHAImageDependencies(t *testing.T) {
	for name, test := range map[string]struct {
		labels                      []string
		build, waitGHA, waitKonflux bool
	}{
		"no consumers": {},
		"GHA consumer": {labels: []string{"e2e-gke-upgrade-tests"}, build: true, waitGHA: true},
		"Konflux consumer": {
			labels: []string{"e2e-nongroovy-tests-gke-konflux", "konflux-build"}, waitKonflux: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			resolved, d, err := openDecision("../test-domains.yaml", "../decision-defaults",
				[]string{"README.md"}, test.labels, false, metadata{})
			require.NoError(t, err)
			assert.Equal(t, test.build, resolved.jobs["build-images"])
			plan := dispatchPlan(dispatchInput{pullRequest: true, enforce: true, labels: test.labels}, &resolved, d)
			assert.Equal(t, fmt.Sprint(test.waitGHA), plan["wait-for-gha-images"])
			assert.Equal(t, fmt.Sprint(test.waitKonflux), plan["wait-for-konflux-images"])
		})
	}
}

func TestChangedFiles(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CI_SELECT_ROOT", root)
	git := func(args ...string) string {
		t.Helper()
		command := exec.CommandContext(t.Context(), "git", append([]string{
			"-C", root, "-c", "core.hooksPath=/dev/null", "-c", "commit.gpgsign=false",
		}, args...)...)
		text, err := command.CombinedOutput()
		require.NoError(t, err, string(text))
		return string(bytes.TrimSpace(text))
	}
	git("init")
	git("-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "--allow-empty", "-m", "base")
	base := git("rev-parse", "HEAD")
	files, unknown := changedFiles(t.Context(), base, "HEAD")
	assert.False(t, unknown)
	assert.Empty(t, files)
	writeTestFile(t, filepath.Join(root, "with space.go"), "package example\n", 0o600)
	git("add", "with space.go")
	git("-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-m", "change")
	files, unknown = changedFiles(t.Context(), base, "HEAD")
	assert.False(t, unknown)
	assert.Equal(t, []string{"with space.go"}, files)
	_, unknown = changedFiles(t.Context(), "missing", "HEAD")
	assert.True(t, unknown)
}

func TestGHADispatchCI(t *testing.T) {
	t.Setenv("CI_SELECT_ROOT", "../..")
	t.Setenv("GITHUB_EVENT_NAME", "pull_request")
	t.Setenv("CI_PR", "23166")
	t.Setenv("CI_REPO", "stackrox/stackrox")
	t.Setenv("CI_BASE", "missing")
	t.Setenv("CI_IS_FORK", "true")
	t.Setenv("GITHUB_OUTPUT", "")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"name":"ci-dispatcher-enforce"}]`)
	}))
	defer server.Close()
	t.Setenv("GITHUB_API_URL", server.URL)
	var output bytes.Buffer
	require.NoError(t, ghaDispatchCI(t.Context(), &output, &bytes.Buffer{}))
	assert.Contains(t, output.String(), `"e2e-nongroovy-tests-gke":false`)
	assert.Contains(t, output.String(), "wait-for-gha-images=false")
}

func TestGHADispatchCILabelFailure(t *testing.T) {
	for name, test := range map[string]struct {
		labels, fork, draft, uiOnly        string
		nongroovy, byodb, konflux, waitGHA bool
	}{
		"ready PR without labels": {nongroovy: true, waitGHA: true},
		"draft without labels":    {draft: "true"},
		"UI only":                 {uiOnly: "true", waitGHA: true},
		"explicit job on draft": {
			labels: "ci-dispatcher-enforce,e2e-byodb-tests", draft: "true",
			byodb: true, waitGHA: true,
		},
		"Konflux label": {
			labels: "konflux-build", nongroovy: true, konflux: true, waitGHA: true,
		},
		"stale enforcement label": {
			labels: "ci-dispatcher-enforce", nongroovy: true, waitGHA: true,
		},
		"fork with explicit job": {
			labels: "ci-dispatcher-enforce,e2e-byodb-tests,konflux-build", fork: "true",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("CI_SELECT_ROOT", t.TempDir())
			t.Setenv("GITHUB_EVENT_NAME", "pull_request")
			t.Setenv("CI_PR", "23403")
			t.Setenv("CI_REPO", "stackrox/stackrox")
			t.Setenv("CI_IS_FORK", test.fork)
			t.Setenv("CI_IS_DRAFT", test.draft)
			t.Setenv("CI_UI_ONLY", test.uiOnly)
			t.Setenv("PR_LABELS", test.labels)
			outputPath := filepath.Join(t.TempDir(), "output")
			t.Setenv("GITHUB_OUTPUT", outputPath)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/repos/stackrox/stackrox/issues/23403/labels", r.URL.Path)
				w.WriteHeader(http.StatusForbidden)
			}))
			defer server.Close()
			t.Setenv("GITHUB_API_URL", server.URL)
			var output, diagnostic bytes.Buffer
			require.NoError(t, ghaDispatchCI(t.Context(), &output, &diagnostic))
			assert.Contains(t, diagnostic.String(), "using event labels and existing dispatch rules with enforcement disabled")
			assert.NotContains(t, diagnostic.String(), "CI decision unreadable")
			assert.Contains(t, output.String(), fmt.Sprintf(`"e2e-nongroovy-tests-gke":%t`, test.nongroovy))
			assert.Contains(t, output.String(), fmt.Sprintf(`"e2e-byodb-tests":%t`, test.byodb))
			assert.Contains(t, output.String(), fmt.Sprintf(`"e2e-qa-tests-gke-konflux":%t`, test.konflux))
			assert.Contains(t, output.String(), fmt.Sprintf("wait-for-gha-images=%t", test.waitGHA))
			assert.Contains(t, output.String(), fmt.Sprintf("wait-for-konflux-images=%t", test.konflux))
			written, err := os.ReadFile(outputPath)
			require.NoError(t, err)
			assert.Equal(t, output.String(), string(written))
		})
	}
}
