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
	rules, err := os.ReadFile("../test-domains.yaml")
	require.NoError(t, err)
	cases := map[string]struct {
		mutate  func(t *testing.T)
		preview bool
		job     string
		want    string
		log     string
	}{
		"should apply an enforced skip":           {want: "skip", log: "applied=skip"},
		"should only preview without enforcement": {preview: true, want: "run", log: "planned=skip; applied=run"},
		"should run unknown jobs":                 {job: "not-enrolled", want: "run"},
		"should run with missing refs": {
			mutate: func(t *testing.T) { writeEvent(t, root, "missing", head, true) }, want: "run", log: "collect diff",
		},
		"should run with empty diff": {
			mutate: func(t *testing.T) { writeEvent(t, root, head, head, true) }, want: "run", log: "empty diff",
		},
		"should run with malformed rules": {
			mutate: func(t *testing.T) { writeTestFile(t, filepath.Join(root, "ci/test-domains.yaml"), "[") },
			want:   "run", log: "selection unavailable",
		},
		"should run with missing rules": {
			mutate: func(t *testing.T) { require.NoError(t, os.Remove(filepath.Join(root, "ci/test-domains.yaml"))) },
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
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			writeTestFile(t, filepath.Join(root, "ci/test-domains.yaml"), string(rules))
			writeEvent(t, root, base, head, !c.preview)
			// Enforcement comes from the event snapshot, including label removal.
			t.Setenv("CI_ENFORCE", "true")
			t.Setenv("PR_LABELS", enforceLabel)
			t.Setenv("CI_SELECT_ROOT", root)
			t.Setenv("GITHUB_OUTPUT", filepath.Join(t.TempDir(), "output"))
			t.Setenv("GITHUB_STEP_SUMMARY", filepath.Join(t.TempDir(), "summary"))
			if c.mutate != nil {
				c.mutate(t)
			}
			jobs := []string{c.job}
			if c.job == "" {
				jobs = []string{"go", "go-postgres", "go-bench", "sensor-integration-tests"}
			}
			for _, job := range jobs {
				var output, diagnostic bytes.Buffer
				require.NoError(t, run(t.Context(), []string{"gha", job}, &output, &diagnostic))
				assert.Equal(t, "action="+c.want+"\n", output.String())
				assert.Contains(t, diagnostic.String(), c.log)
			}
			data, err := os.ReadFile(os.Getenv("GITHUB_OUTPUT"))
			require.NoError(t, err)
			assert.Equal(t, strings.Repeat("action="+c.want+"\n", len(jobs)), string(data))
			data, err = os.ReadFile(os.Getenv("GITHUB_STEP_SUMMARY"))
			require.NoError(t, err)
			assert.Contains(t, string(data), "## CI selection")
			if c.job == "" {
				t.Setenv("GITHUB_OUTPUT", filepath.Join(t.TempDir(), "planner-output"))
				var output, diagnostic bytes.Buffer
				require.NoError(t, run(t.Context(), []string{"gha"}, &output, &diagnostic))
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
				assert.Contains(t, output.String(), c.log)
			}
		})
	}
	t.Run("should print a preview without gating validation", func(t *testing.T) {
		writeTestFile(t, filepath.Join(root, "ci/test-domains.yaml"), string(rules))
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

func TestGateRecovery(t *testing.T) {
	t.Parallel()
	wrapper, err := os.ReadFile("gha-gate.sh")
	require.NoError(t, err)
	for name, c := range map[string]struct{ script, build, want string }{
		"should run when executable is missing": {want: "run"},
		"should run when executable crashes":    {script: "exit 2", want: "run"},
		"should override a partial skip on failure": {
			script: "echo action=skip >>\"$GITHUB_OUTPUT\"\nexit 2", want: "run",
		},
		"should avoid stale executable after a build failure": {
			script: "exit 99", build: "failure", want: "run",
		},
		"should retain a successful skip": {script: "echo action=skip >>\"$GITHUB_OUTPUT\"", want: "skip"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			path := writeTestFile(t, filepath.Join(root, "ci/select/gha-gate.sh"), string(wrapper))
			if c.script != "" {
				binary := writeTestFile(t, filepath.Join(root, "ci/select/bin/ci-select"), "#!/usr/bin/env bash\n"+c.script+"\n")
				require.NoError(t, os.Chmod(binary, 0o700))
			}
			out := filepath.Join(root, "output")
			cmd := exec.CommandContext(t.Context(), "bash", path, "go")
			cmd.Env = append(os.Environ(), "GITHUB_OUTPUT="+out, "GITHUB_STEP_SUMMARY="+filepath.Join(root, "summary"),
				"CI_SELECT_BUILD_OUTCOME="+c.build)
			if c.build == "" {
				cmd.Env = append(cmd.Env, "CI_SELECT_BUILD_OUTCOME=success")
			}
			data, err := cmd.CombinedOutput()
			require.NoError(t, err, "%s", data)
			data, err = os.ReadFile(out)
			require.NoError(t, err)
			assert.True(t, strings.HasSuffix(string(data), "action="+c.want+"\n"), "%s", data)
			if c.build == "failure" {
				assert.Contains(t, string(data), "action=run")
			}
		})
	}
}

func TestPlannerRecovery(t *testing.T) {
	t.Parallel()
	wrapper, err := os.ReadFile("gha-gate.sh")
	require.NoError(t, err)
	for name, script := range map[string]string{
		"missing binary":             "",
		"crash after partial output": "echo 'decisions={\"go\":\"skip\"}' >>\"$GITHUB_OUTPUT\"\nexit 2",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			wrapperPath := writeTestFile(t, filepath.Join(root, "ci/select/gha-gate.sh"), string(wrapper))
			if script != "" {
				binary := writeTestFile(t, filepath.Join(root, "ci/select/bin/ci-select"), "#!/usr/bin/env bash\n"+script+"\n")
				require.NoError(t, os.Chmod(binary, 0o700))
			}
			output := filepath.Join(root, "output")
			cmd := exec.CommandContext(t.Context(), "bash", wrapperPath)
			cmd.Env = append(os.Environ(), "GITHUB_OUTPUT="+output, "GITHUB_STEP_SUMMARY="+filepath.Join(root, "summary"), "CI_SELECT_BUILD_OUTCOME=success")
			data, err := cmd.CombinedOutput()
			require.NoError(t, err, "%s", data)
			data, err = os.ReadFile(output)
			require.NoError(t, err)
			assert.True(t, strings.HasSuffix(string(data), "decisions={}\n"), "%s", data)
		})
	}
}

func TestConsumeSharedDecision(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct{ result, decisions, target, want string }{
		"explicit skip":          {"success", `{"go":"skip","go-bench":"run"}`, "go", "skip"},
		"explicit run":           {"success", `{"go":"run"}`, "go", "run"},
		"hyphenated target":      {"success", `{"sensor-integration-tests":"skip"}`, "sensor-integration-tests", "skip"},
		"missing map":            {"success", "", "go", "run"},
		"empty map":              {"success", `{}`, "go", "run"},
		"missing target":         {"success", `{"go-bench":"skip"}`, "go", "run"},
		"invalid JSON":           {"success", `{`, "go", "run"},
		"scalar":                 {"success", `"skip"`, "go", "run"},
		"array":                  {"success", `["skip"]`, "go", "run"},
		"null":                   {"success", `null`, "go", "run"},
		"invalid decision":       {"success", `{"go":"Skip"}`, "go", "run"},
		"invalid member":         {"success", `{"go":"skip","other":true}`, "go", "run"},
		"multiple documents":     {"success", `{"go":"skip"} {"go":"skip"}`, "go", "run"},
		"failed planner":         {"failure", `{"go":"skip"}`, "go", "run"},
		"skipped planner":        {"skipped", `{"go":"skip"}`, "go", "run"},
		"cancelled planner":      {"cancelled", `{"go":"skip"}`, "go", "run"},
		"missing planner result": {"", `{"go":"skip"}`, "go", "run"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			output := filepath.Join(t.TempDir(), "output")
			cmd := exec.CommandContext(t.Context(), "bash", "gha-consume.sh", c.target)
			cmd.Env = append(os.Environ(), "GITHUB_OUTPUT="+output, "CI_SELECT_PLANNER_RESULT="+c.result, "CI_SELECT_DECISIONS="+c.decisions)
			data, err := cmd.CombinedOutput()
			require.NoError(t, err, "%s", data)
			data, err = os.ReadFile(output)
			require.NoError(t, err)
			assert.Equal(t, "action="+c.want+"\n", string(data))
		})
	}
}
