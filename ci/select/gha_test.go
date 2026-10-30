package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "git", append([]string{"-C", root}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
	data, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", data)
	return strings.TrimSpace(string(data))
}

func testRepo(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	testGit(t, root, "init", "--initial-branch=main")
	testGit(t, root, "config", "user.name", "CI selection tests")
	testGit(t, root, "config", "user.email", "ci-select@example.invalid")
	testGit(t, root, "config", "core.excludesFile", "/dev/null")
	writeTestFile(t, filepath.Join(root, "README.md"), "base\n")
	testGit(t, root, "add", "--", "README.md")
	testGit(t, root, "commit", "-m", "Base fixture")
	return root, testGit(t, root, "rev-parse", "HEAD")
}

func TestChangedFiles(t *testing.T) {
	t.Parallel()
	root, _ := testRepo(t)
	path := "central/testdata/a\nfixture.md"
	writeTestFile(t, filepath.Join(root, path), "fixture\n")
	testGit(t, root, "add", "--", path)
	testGit(t, root, "commit", "-m", "Backend fixture")
	testGit(t, root, "switch", "-c", "pr")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "docs"), 0o755))
	require.NoError(t, os.Rename(filepath.Join(root, path), filepath.Join(root, "docs/moved.md")))
	testGit(t, root, "add", "--", path, "docs/moved.md")
	testGit(t, root, "commit", "-m", "Rename fixture")
	head := testGit(t, root, "rev-parse", "HEAD")
	testGit(t, root, "switch", "main")
	writeTestFile(t, filepath.Join(root, "base-only.go"), "base advanced\n")
	testGit(t, root, "add", "--", "base-only.go")
	testGit(t, root, "commit", "-m", "Advance base")
	base := testGit(t, root, "rev-parse", "HEAD")
	files, err := changedFiles(t.Context(), root, base, head)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{path, "docs/moved.md"}, files)
	files, err = changedFiles(t.Context(), root, head, head)
	require.NoError(t, err)
	assert.Empty(t, files)
	for _, ref := range []string{"", "missing-ref", "--stat"} {
		_, err := changedFiles(t.Context(), root, ref, head)
		require.Error(t, err)
	}
}

// testSourceRepo builds a base branch and a PR branch whose merge base is the
// first commit, with the base branch advanced after the fork point.
func testSourceRepo(t *testing.T) (root, base, head string) {
	t.Helper()
	root = t.TempDir()
	testGit(t, root, "init", "--initial-branch=main")
	testGit(t, root, "config", "user.name", "CI selection tests")
	testGit(t, root, "config", "user.email", "ci-select@example.invalid")
	testGit(t, root, "config", "core.excludesFile", "/dev/null")
	writeTestFile(t, filepath.Join(root, "README.md"), "base\n")
	testGit(t, root, "add", "--", "README.md")
	testGit(t, root, "commit", "-m", "Merge base")
	testGit(t, root, "switch", "-c", "pr")
	writeTestFile(t, filepath.Join(root, "docs/guide.md"), "docs\n")
	testGit(t, root, "add", "--", "docs/guide.md")
	testGit(t, root, "commit", "-m", "PR change")
	head = testGit(t, root, "rev-parse", "HEAD")
	testGit(t, root, "switch", "main")
	writeTestFile(t, filepath.Join(root, "base-only.go"), "base advanced\n")
	testGit(t, root, "add", "--", "base-only.go")
	testGit(t, root, "commit", "-m", "Advance base")
	base = testGit(t, root, "rev-parse", "HEAD")
	return root, base, head
}

// testPlannerRepo creates an empty repository that fetches from source, like a
// depth-1 workflow checkout that holds neither diff endpoint.
func testPlannerRepo(t *testing.T, source string) string {
	t.Helper()
	root := t.TempDir()
	testGit(t, root, "init", "--initial-branch=main")
	testGit(t, root, "remote", "add", "origin", "file://"+source)
	return root
}

func TestChangedFilesFetchesMissingHistory(t *testing.T) {
	t.Parallel()
	source, base, head := testSourceRepo(t)
	// Local remotes reject unadvertised SHAs unless the source opts in,
	// mirroring the server-side setting that makes GitHub SHA fetches work.
	testGit(t, source, "config", "uploadpack.allowAnySHA1InWant", "true")
	planner := testPlannerRepo(t, source)
	files, err := changedFiles(t.Context(), planner, base, head)
	require.NoError(t, err)
	assert.Equal(t, []string{"docs/guide.md"}, files)
}

func TestChangedFilesWithCompleteHistoryDoesNotFetch(t *testing.T) {
	t.Parallel()
	source, base, head := testSourceRepo(t)
	planner := t.TempDir()
	testGit(t, planner, "clone", "--quiet", "file://"+source, planner)
	testGit(t, planner, "config", "core.excludesFile", "/dev/null")
	// A broken remote proves the complete clone never fetches.
	testGit(t, planner, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "missing-remote"))
	files, err := changedFiles(t.Context(), planner, base, head)
	require.NoError(t, err)
	assert.Equal(t, []string{"docs/guide.md"}, files)
}

func TestChangedFilesDisconnectedHistoriesFailClosed(t *testing.T) {
	t.Parallel()
	source := t.TempDir()
	testGit(t, source, "init", "--initial-branch=main")
	testGit(t, source, "config", "user.name", "CI selection tests")
	testGit(t, source, "config", "user.email", "ci-select@example.invalid")
	testGit(t, source, "config", "core.excludesFile", "/dev/null")
	writeTestFile(t, filepath.Join(source, "a.txt"), "a\n")
	testGit(t, source, "add", "--", "a.txt")
	testGit(t, source, "commit", "-m", "Root one")
	testGit(t, source, "switch", "--orphan", "island")
	writeTestFile(t, filepath.Join(source, "b.txt"), "b\n")
	testGit(t, source, "add", "--", "b.txt")
	testGit(t, source, "commit", "-m", "Root two")
	main := testGit(t, source, "rev-parse", "main")
	island := testGit(t, source, "rev-parse", "island")
	testGit(t, source, "config", "uploadpack.allowAnySHA1InWant", "true")
	planner := testPlannerRepo(t, source)
	_, err := changedFiles(t.Context(), planner, main, island)
	require.ErrorContains(t, err, "no merge base")
}

func TestChangedFilesWithUnfetchableRemoteFallsBack(t *testing.T) {
	t.Parallel()
	_, base, head := testSourceRepo(t)
	planner := testPlannerRepo(t, filepath.Join(t.TempDir(), "missing-remote"))
	_, err := changedFiles(t.Context(), planner, base, head)
	require.Error(t, err)
}

func writeEvent(t *testing.T, root, base, head string, enforce bool) {
	t.Helper()
	labels := []map[string]string{}
	if enforce {
		labels = append(labels, map[string]string{"name": enforceLabel})
	}
	data, err := json.Marshal(map[string]any{"pull_request": map[string]any{
		"base": map[string]string{"sha": base}, "head": map[string]string{"sha": head}, "labels": labels,
	}})
	require.NoError(t, err)
	t.Setenv("GITHUB_EVENT_NAME", "pull_request")
	t.Setenv("GITHUB_EVENT_PATH", writeTestFile(t, filepath.Join(root, "event.json"), string(data)))
}

func TestGHA(t *testing.T) {
	root, base := testRepo(t)
	writeTestFile(t, filepath.Join(root, "docs/guide.md"), "docs\n")
	testGit(t, root, "add", "--", "docs/guide.md")
	testGit(t, root, "commit", "-m", "Docs fixture")
	head := testGit(t, root, "rev-parse", "HEAD")
	rules, err := os.ReadFile("../test-selection-rules.yaml")
	require.NoError(t, err)
	cases := map[string]struct {
		mutate  func(t *testing.T)
		preview bool
		want    string
		log     string
	}{
		"should apply an enforced skip":           {want: "skip", log: "applied=skip"},
		"should only preview without enforcement": {preview: true, want: "run", log: "planned=skip; applied=run"},
		"should run with missing refs": {
			mutate: func(t *testing.T) { writeEvent(t, root, "missing", head, true) }, want: "run", log: "collect diff",
		},
		"should run with empty diff": {
			mutate: func(t *testing.T) { writeEvent(t, root, head, head, true) }, want: "run", log: "empty diff",
		},
		"should run with malformed rules": {
			mutate: func(t *testing.T) { writeTestFile(t, filepath.Join(root, "ci/test-selection-rules.yaml"), "[") },
			want:   "run", log: "selection unavailable",
		},
		"should run with missing rules": {
			mutate: func(t *testing.T) { require.NoError(t, os.Remove(filepath.Join(root, "ci/test-selection-rules.yaml"))) },
			want:   "run", log: "read rules",
		},
		"should run with malformed event": {
			mutate: func(t *testing.T) { writeTestFile(t, filepath.Join(root, "event.json"), "[") },
			want:   "run", log: "decode PR event",
		},
		"should run with missing event": {
			mutate: func(t *testing.T) { t.Setenv("GITHUB_EVENT_PATH", filepath.Join(root, "missing.json")) },
			want:   "run", log: "read PR event",
		},
		"should run without a PR snapshot": {
			mutate: func(t *testing.T) { writeTestFile(t, filepath.Join(root, "event.json"), "{}") },
			want:   "run", log: "no pull request",
		},
		"should preserve push behavior": {
			mutate: func(t *testing.T) { t.Setenv("GITHUB_EVENT_NAME", "push") }, want: "run", log: "only to pull requests",
		},
	}
	jobs := []string{"go", "go-postgres", "go-bench", "sensor-integration-tests"}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			writeTestFile(t, filepath.Join(root, "ci/test-selection-rules.yaml"), string(rules))
			writeEvent(t, root, base, head, !c.preview)
			t.Setenv("CI_SELECT_ROOT", root)
			t.Setenv("GITHUB_OUTPUT", filepath.Join(t.TempDir(), "output"))
			t.Setenv("GITHUB_STEP_SUMMARY", filepath.Join(t.TempDir(), "summary"))
			if c.mutate != nil {
				c.mutate(t)
			}
			var output, diagnostic bytes.Buffer
			require.NoError(t, run(t.Context(), []string{"gha"}, &output, &diagnostic))
			assert.Contains(t, output.String(), c.log)
			data, err := os.ReadFile(os.Getenv("GITHUB_OUTPUT"))
			require.NoError(t, err)
			var decisions map[string]string
			require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(string(data), "decisions="))), &decisions))
			for _, job := range jobs {
				action := decisions[job]
				if action == "" {
					action = "run"
				}
				assert.Equal(t, c.want, action, job)
			}
			data, err = os.ReadFile(os.Getenv("GITHUB_STEP_SUMMARY"))
			require.NoError(t, err)
			assert.Contains(t, string(data), "## CI selection")
		})
	}
	t.Run("should print a preview without gating validation", func(t *testing.T) {
		writeTestFile(t, filepath.Join(root, "ci/test-selection-rules.yaml"), string(rules))
		writeEvent(t, root, base, head, false)
		t.Setenv("CI_SELECT_ROOT", root)
		t.Setenv("GITHUB_OUTPUT", filepath.Join(t.TempDir(), "output"))
		t.Setenv("GITHUB_STEP_SUMMARY", filepath.Join(t.TempDir(), "summary"))
		var output, diagnostic bytes.Buffer
		require.NoError(t, run(t.Context(), []string{"gha"}, &output, &diagnostic))
		assert.Contains(t, output.String(), "enforce=false")
		assert.Contains(t, output.String(), "sensor-integration-tests skip")
		assert.Empty(t, diagnostic.String())
		_, err := os.Stat(os.Getenv("GITHUB_OUTPUT"))
		require.NoError(t, err)
		data, err := os.ReadFile(os.Getenv("GITHUB_OUTPUT"))
		require.NoError(t, err)
		assert.Contains(t, string(data), `"go":"run"`)
	})
}
