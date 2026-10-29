//go:build integration

package main

import (
	"encoding/json"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScriptPlannerRecovery(t *testing.T) {
	t.Parallel()
	wrapper, err := os.ReadFile("gha-gate.sh")
	require.NoError(t, err)
	const skip = `printf 'decisions={"example-suite":"skip"}\n' >>"$GITHUB_OUTPUT"`
	for name, c := range map[string]struct {
		script, build, want string
		wantCalled          bool
	}{
		"should run when the executable is missing": {build: "success", want: "{}"},
		"should discard partial skips when the executable crashes": {
			script: skip + "\nexit 2", build: "success", want: "{}", wantCalled: true,
		},
		"should not execute a stale binary after a failed build": {
			script: skip, build: "failure", want: "{}",
		},
		"should preserve a successful decision": {
			script: skip, build: "success", want: `{"example-suite":"skip"}`, wantCalled: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			gate := writeTestFile(t, filepath.Join(root, "ci/select/gha-gate.sh"), string(wrapper))
			if c.script != "" {
				binary := writeTestFile(t, filepath.Join(root, "ci/select/bin/ci-select"),
					"#!/usr/bin/env bash\ntouch \"$CI_SELECT_ROOT/called\"\n"+c.script+"\n")
				require.NoError(t, os.Chmod(binary, 0o700))
			}
			outputs := runSelectionScript(t, map[string]string{"CI_SELECT_BUILD_OUTCOME": c.build}, "bash", gate)
			assert.Equal(t, c.want, outputs["decisions"])
			_, err := os.Stat(filepath.Join(root, "called"))
			if c.wantCalled {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, os.ErrNotExist)
			}
		})
	}
}

func TestScriptConsumeDecision(t *testing.T) {
	t.Parallel()
	const skip = `{"example-suite":"skip"}`
	for name, c := range map[string]struct{ result, decisions, want string }{
		"should apply an explicit skip":                    {"success", skip, "skip"},
		"should preserve an explicit run":                  {"success", `{"example-suite":"run"}`, "run"},
		"should run when the target is missing":            {"success", `{}`, "run"},
		"should run when output is missing":                {"success", "", "run"},
		"should run when JSON is malformed":                {"success", "{", "run"},
		"should run when JSON is not an object":            {"success", `[]`, "run"},
		"should reject a map containing invalid decisions": {"success", `{"example-suite":"skip","other":false}`, "run"},
		"should reject multiple JSON documents":            {"success", skip + " {}", "run"},
		"should ignore a skip from a failed planner":       {"failure", skip, "run"},
		"should ignore a skip from a skipped planner":      {"skipped", skip, "run"},
		"should ignore a skip from a cancelled planner":    {"cancelled", skip, "run"},
		"should run when the planner result is missing":    {"", skip, "run"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			outputs := runSelectionScript(t, map[string]string{
				"CI_SELECT_PLANNER_RESULT": c.result,
				"CI_SELECT_DECISIONS":      c.decisions,
			}, "bash", "gha-consume.sh", "example-suite")
			assert.Equal(t, c.want, outputs["action"])
		})
	}
}

func TestScriptE2EDispatch(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	gh := writeTestFile(t, filepath.Join(root, "gh"), "#!/usr/bin/env bash\nprintf '%s' \"$TEST_PR_LABELS\"\n")
	require.NoError(t, os.Chmod(gh, 0o700))
	env := map[string]string{
		"PATH":                     root + string(os.PathListSeparator) + os.Getenv("PATH"),
		"TEST_PR_LABELS":           "",
		"input__event_name":        "pull_request",
		"input__is_fork":           "false",
		"input__is_draft":          "false",
		"input__ui_only_change":    "false",
		"input__ci_decisions":      "",
		"input__ci_planner_result": "success",
		"input__repo":              "example/repo",
		"input__pr_number":         "1",
	}
	baseline := runSelectionScript(t, env, "python3", "e2e_dispatch.py")
	var baselineActions map[string]string
	var baselineJobs map[string]bool
	require.NoError(t, json.Unmarshal([]byte(baseline["ci-actions"]), &baselineActions))
	require.NoError(t, json.Unmarshal([]byte(baseline["jobs"]), &baselineJobs))
	require.NotEmpty(t, baselineActions)

	allRun, allSkip := make(map[string]string), make(map[string]string)
	allJobs, noJobs := make(map[string]bool), make(map[string]bool)
	for target := range baselineActions {
		allRun[target], allSkip[target] = "run", "skip"
		allJobs[target], noJobs[target] = true, false
	}
	assert.Equal(t, allRun, baselineActions)
	skipJSON, err := json.Marshal(allSkip)
	require.NoError(t, err)

	const optionalTarget = "e2e-byodb-tests"
	require.Contains(t, baselineJobs, optionalTarget)
	require.False(t, baselineJobs[optionalTarget])
	partialActions, partialJobs := maps.Clone(allRun), maps.Clone(baselineJobs)
	partialActions[optionalTarget], partialJobs[optionalTarget] = "skip", true

	for name, c := range map[string]struct {
		env        map[string]string
		actions    map[string]string
		jobs       map[string]bool
		skipImages string
	}{
		"should admit skipped checks on a draft without optional labels": {
			env:     map[string]string{"input__ci_decisions": string(skipJSON), "input__is_draft": "true"},
			actions: allSkip, jobs: allJobs, skipImages: "true",
		},
		"should apply a partial selection without changing other eligibility": {
			env:     map[string]string{"input__ci_decisions": `{"e2e-byodb-tests":"skip"}`},
			actions: partialActions, jobs: partialJobs, skipImages: "false",
		},
		"should retain normal eligibility when decisions are malformed": {
			env:     map[string]string{"input__ci_decisions": "{"},
			actions: allRun, jobs: baselineJobs, skipImages: "false",
		},
		"should ignore stale skips when the planner fails": {
			env:     map[string]string{"input__ci_decisions": string(skipJSON), "input__ci_planner_result": "failure"},
			actions: allRun, jobs: baselineJobs, skipImages: "false",
		},
		"should not admit a fork through a skip decision": {
			env:     map[string]string{"input__ci_decisions": string(skipJSON), "input__is_fork": "true"},
			actions: allRun, jobs: noJobs, skipImages: "false",
		},
		"should ignore skip decisions on a push": {
			env:     map[string]string{"input__ci_decisions": string(skipJSON), "input__event_name": "push"},
			actions: allRun, jobs: allJobs, skipImages: "false",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			inputs := maps.Clone(env)
			maps.Copy(inputs, c.env)
			outputs := runSelectionScript(t, inputs, "python3", "e2e_dispatch.py")
			var actions map[string]string
			var jobs map[string]bool
			require.NoError(t, json.Unmarshal([]byte(outputs["ci-actions"]), &actions))
			require.NoError(t, json.Unmarshal([]byte(outputs["jobs"]), &jobs))
			assert.Equal(t, c.actions, actions)
			assert.Equal(t, c.jobs, jobs)
			assert.Equal(t, c.skipImages, outputs["skip-gha-images"])
			assert.Equal(t, c.skipImages, outputs["skip-konflux-images"])
		})
	}
}

// runSelectionScript reads the last value for each output, matching GitHub's output files.
func runSelectionScript(t *testing.T, env map[string]string, executable string, args ...string) map[string]string {
	t.Helper()
	root := t.TempDir()
	output := filepath.Join(root, "output")
	cmd := exec.CommandContext(t.Context(), executable, args...)
	cmd.Env = os.Environ()
	for key, value := range env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	cmd.Env = append(cmd.Env, "GITHUB_OUTPUT="+output, "GITHUB_STEP_SUMMARY="+filepath.Join(root, "summary"))
	log, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", log)
	data, err := os.ReadFile(output)
	require.NoError(t, err)
	outputs := make(map[string]string)
	for line := range strings.Lines(string(data)) {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		require.True(t, ok, "invalid script output: %q", line)
		outputs[key] = value
	}
	return outputs
}
