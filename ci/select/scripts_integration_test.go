//go:build integration

package main

import (
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
