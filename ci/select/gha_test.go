package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestGHACommentOutput(t *testing.T) {
	for name, test := range map[string]struct {
		args        []string
		invalidPath bool
		wantComment bool
	}{
		"should write comment when printing the plan": {wantComment: true},
		"should accept an empty job argument":         {args: []string{""}, wantComment: true},
		"should not write comment when gating a job":  {args: []string{"go"}},
		"should report comment write failures":        {invalidPath: true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("CI_SELECT_ROOT", "../..")
			t.Setenv("CI_BASE", "missing-ref")
			t.Setenv("CI_HEAD", "HEAD")
			t.Setenv("CI_COMMIT", "0123456789abcdef")
			t.Setenv("CI_REPO", "stackrox/stackrox")
			t.Setenv("CI_ENFORCE", "true")
			t.Setenv("GITHUB_OUTPUT", "")
			t.Setenv("GITHUB_STEP_SUMMARY", "")
			path := filepath.Join(t.TempDir(), "comment.md")
			if test.invalidPath {
				path = filepath.Join(path, "missing", "comment.md")
			} else if test.wantComment {
				writeTestFile(t, path, "stale decision", 0o600)
			}
			t.Setenv("CI_COMMENT_OUT", path)
			var output bytes.Buffer
			err := ghaCI(t.Context(), test.args, &output, &bytes.Buffer{})
			if test.invalidPath {
				require.ErrorContains(t, err, "write decision comment")
				return
			}
			require.NoError(t, err)
			if !test.wantComment {
				_, err := os.Stat(path)
				assert.ErrorIs(t, err, os.ErrNotExist)
				return
			}
			text, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Contains(t, string(text), "[0123456](https://github.com/stackrox/stackrox/commit/0123456789abcdef)")
			assert.Contains(t, string(text), "<summary>Resolver log</summary>")
			assert.NotContains(t, string(text), "stale decision")
			assert.Contains(t, output.String(), "Decision:\n")
			assert.Contains(t, output.String(), "Log:\n")
		})
	}
}

func TestPlanComment(t *testing.T) {
	for name, test := range map[string]struct {
		commit, shortSHA, decision, log, mode string
		enforce                               bool
	}{
		"should show the enforced decision and collapse the log": {
			commit: "abcdef0123456789", shortSHA: "abcdef0", decision: "go\nstyle\n", log: "matched Go rule\n",
			enforce: true, mode: "Selection is enforced",
		},
		"should explain preview mode when no jobs are selected": {
			commit: "123456789abcdef0", shortSHA: "1234567", decision: "# no job runs\n", log: "all jobs skipped\n",
			mode: "jobs run regardless of this decision",
		},
		"should preserve log text containing HTML and Markdown": {
			commit: "abc", shortSHA: "abc", decision: "go\n", log: "</pre></details><details open>\n```\nlabel: a & b\n",
			mode: "Selection is in preview mode",
		},
	} {
		t.Run(name, func(t *testing.T) {
			comment := planComment(result{decision: test.decision, log: test.log}, test.enforce,
				metadata{repo: "stackrox/stackrox", commit: test.commit})
			assert.Contains(t, comment, fmt.Sprintf("[%s](https://github.com/stackrox/stackrox/commit/%s)", test.shortSHA, test.commit))
			assert.Contains(t, comment, test.mode)
			visible, collapsed, found := strings.Cut(comment, "<details>\n<summary>Resolver log</summary>")
			require.True(t, found)
			assert.Contains(t, visible, "<pre>"+html.EscapeString(test.decision)+"</pre>")
			assert.Contains(t, collapsed, "<pre>"+html.EscapeString(test.log)+"</pre>")
			assert.Equal(t, 1, strings.Count(comment, "<details>"))
			assert.Equal(t, 1, strings.Count(comment, "</details>"))
			assert.NotContains(t, comment, "<details open>")
		})
	}
}

func TestDispatchPlan(t *testing.T) {
	d := defaults{actions: map[string]string{"wait-for-images": "run", "wait-for-konflux-images": "run"}}
	for _, job := range dispatchJobs {
		d.actions[job.name] = "run"
	}
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
		"enforced consumer without dependency vote": {
			input:    dispatchInput{pullRequest: true, enforce: true},
			resolved: &result{jobs: map[string]bool{"e2e-nongroovy-tests-gke": true}}, nongroovy: true, waitGHA: true,
		},
		"enforced Konflux consumer without dependency vote": {
			input:    dispatchInput{pullRequest: true, enforce: true, labels: []string{"konflux-build"}},
			resolved: &result{jobs: map[string]bool{"e2e-nongroovy-tests-gke-konflux": true}}, nongroovyKonflux: true, waitKonflux: true,
		},
		"shadow": {input: dispatchInput{pullRequest: true}, resolved: &result{jobs: map[string]bool{}}, nongroovy: true, waitGHA: true},
		"fork":   {input: dispatchInput{pullRequest: true, fork: true, enforce: true}, resolved: &result{jobs: map[string]bool{"e2e-nongroovy-tests-gke": true, "wait-for-images": true}}},
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
		files, labels        []string
		waitGHA, waitKonflux bool
	}{
		"no GHA consumers": {files: []string{"README.md"}},
		"Prow UI consumer": {files: []string{"ui/example.tsx"}},
		"GHA consumer":     {files: []string{"README.md"}, labels: []string{"e2e-gke-upgrade-tests"}, waitGHA: true},
		"Konflux consumer": {
			files: []string{"README.md"}, labels: []string{"e2e-nongroovy-tests-gke-konflux", "konflux-build"}, waitKonflux: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			resolved, d, err := openDecision("../test-domains.yaml", "../decision-defaults",
				test.files, test.labels, false, metadata{})
			require.NoError(t, err)
			assert.True(t, resolved.jobs["build-images"])
			assert.Equal(t, "run", gateAction("build-images", resolved.jobs, d, true, false))
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
	beforeRename := git("rev-parse", "HEAD")
	git("mv", "with space.go", "README.md")
	git("-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-m", "rename")
	files, unknown = changedFiles(t.Context(), beforeRename, "HEAD")
	assert.False(t, unknown)
	assert.ElementsMatch(t, []string{"with space.go", "README.md"}, files)
	resolved, _, err := openDecision("../test-domains.yaml", "../decision-defaults", files, nil, false, metadata{})
	require.NoError(t, err)
	assert.True(t, resolved.jobs["go"])
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
	t.Setenv("PR_LABELS", "ci-dispatcher-enforce")
	var output bytes.Buffer
	require.NoError(t, ghaDispatchCI(t.Context(), &output, &bytes.Buffer{}))
	assert.Contains(t, output.String(), `"e2e-nongroovy-tests-gke":false`)
	assert.Contains(t, output.String(), "wait-for-gha-images=false")
}

func TestGHADispatchCIEventLabels(t *testing.T) {
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
		"enforced with unavailable defaults": {
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
				t.Error("dispatch must use event labels without an API request")
				w.WriteHeader(http.StatusForbidden)
			}))
			defer server.Close()
			t.Setenv("GITHUB_API_URL", server.URL)
			var output, diagnostic bytes.Buffer
			require.NoError(t, ghaDispatchCI(t.Context(), &output, &diagnostic))
			assert.Equal(t, strings.Contains(test.labels, enforceLabel), strings.Contains(diagnostic.String(), "CI decision unreadable"))
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

func TestGHADispatchCIDefaults(t *testing.T) {
	t.Setenv("CI_SELECT_ROOT", "../..")
	t.Setenv("GITHUB_EVENT_NAME", "pull_request")
	t.Setenv("CI_BASE", "missing-ref")
	t.Setenv("CI_HEAD", "HEAD")
	t.Setenv("CI_IS_FORK", "false")
	t.Setenv("GITHUB_OUTPUT", "")
	t.Setenv("PR_LABELS", "ci-dispatcher-enforce")
	var output, diagnostic bytes.Buffer
	require.NoError(t, ghaDispatchCI(t.Context(), &output, &diagnostic))
	assert.Contains(t, diagnostic.String(), "using decision-defaults")
	assert.Contains(t, output.String(), `"e2e-nongroovy-tests-gke":true`)
	assert.Contains(t, output.String(), `"e2e-qa-tests-gke":false`)
	assert.Contains(t, output.String(), "wait-for-gha-images=true")
}
