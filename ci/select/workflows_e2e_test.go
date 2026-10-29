package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func e2eTargets() []string {
	return []string{
		"e2e-byodb-tests", "e2e-db-backup-restore-tests-gke", "e2e-db-backup-restore-tests-gke-konflux",
		"e2e-gke-upgrade-tests", "e2e-nongroovy-tests-gke", "e2e-nongroovy-tests-gke-konflux",
		"e2e-qa-tests-gke", "e2e-qa-tests-gke-konflux", "e2e-qa-tests-ocp-fips-konflux",
		"e2e-ui-tests-gke", "e2e-ui-tests-gke-konflux",
	}
}

func TestE2ESelectionForwarding(t *testing.T) {
	t.Parallel()
	w := readWorkflow(t, "../../.github/workflows/e2e-dispatch.yaml")
	assert.Equal(t, "./.github/workflows/ci-selection.yaml", w.Jobs["ci-selection"].Uses)
	for _, name := range []string{"ci-selection", "detect-changes", "should-dispatch"} {
		for _, guard := range []string{"github.event_name != 'pull_request'", "github.event.action != 'labeled'", "github.event.action != 'unlabeled'", "github.event.label.name == 'ci-dispatcher-enforce'"} {
			assert.Contains(t, w.Jobs[name].If, guard, "ignore unrelated label events")
		}
	}
	assert.Contains(t, w.On["pull_request"].Types, "labeled")
	assert.Contains(t, w.On["pull_request"].Types, "unlabeled")
	dispatch := w.Jobs["should-dispatch"]
	assert.Contains(t, dispatch.If, "!cancelled()", "failed planning must not suppress dispatch")
	assert.NotContains(t, dispatch.If, "needs.ci-selection.result == 'success'")
	for _, name := range []string{"wait-for-images", "wait-for-konflux-images"} {
		assert.Contains(t, w.Jobs[name].If, "!cancelled()", "planner failure must not suppress image waits for eligible tests")
		assert.Contains(t, w.Jobs[name].If, "needs.should-dispatch.result == 'success'")
	}
	step, _ := findStep(t, dispatch, stepID("determine"))
	assert.Equal(t, "${{ needs.ci-selection.outputs.decisions }}", step.Env["input__ci_decisions"])
	assert.Equal(t, "${{ needs.ci-selection.result }}", step.Env["input__ci_planner_result"])
	for _, target := range e2eTargets() {
		t.Run(target, func(t *testing.T) {
			t.Parallel()
			job, ok := w.Jobs[target]
			require.True(t, ok)
			assert.Contains(t, job.If, "!cancelled()")
			assert.Contains(t, job.If, "needs.should-dispatch.result == 'success'")
			assert.Contains(t, job.If, ")."+target+" == 'skip'", "skip must bypass an unneeded image dependency")
			assert.Contains(t, job.With["ci-action"], ")."+target+" || 'run'")
			assert.Contains(t, job.With["should-run"], ")."+target)
			variant := readWorkflow(t, "../../"+strings.TrimPrefix(job.Uses, "./"))
			assert.Equal(t, "run", variant.On["workflow_call"].Inputs["ci-action"].Default)
			for _, child := range variant.Jobs {
				if child.Uses != "" {
					assert.Equal(t, "${{ inputs.ci-action }}", child.With["ci-action"])
				}
			}
		})
	}
	upgrade := readWorkflow(t, "../../.github/workflows/e2e-gke-upgrade-tests.yaml")
	assert.Contains(t, upgrade.Jobs["gke-upgrade-tests-postgres"].If, "inputs.ci-action == 'skip'")
}

func TestE2EStepGates(t *testing.T) {
	t.Parallel()
	commands := map[string][]string{
		"gke-byodb-qa-e2e-tests":      {"tests/byodb/run.sh"},
		"gke-db-backup-restore-tests": {"db_backup_and_restore_test"},
		"gke-nongroovy-e2e-tests":     {"tests/e2e/run.sh"},
		"qa-e2e-tests":                {"qa-tests-backend/scripts/run-part-1.sh", "qa-tests-backend/scripts/run-part-2.sh"},
		"gke-ui-e2e-tests":            {"tests/e2e/run-ui-e2e.sh"},
		"gke-upgrade-tests-central":   {"tests/upgrade/postgres_run.sh"},
		"gke-upgrade-tests-postgres":  {"tests/upgrade/postgres_upgrade_run.sh"},
		"gke-upgrade-tests-sensor":    {"tests/upgrade/postgres_sensor_run.sh"},
		"check-postgres-upgrade":      {"tests/upgrade/postgres_upgrade_run.sh --check"},
	}
	for _, file := range []string{"e2e-byodb-tests", "e2e-gke-upgrade-tests", "e2e-db-backup-restore-tests", "e2e-nongroovy-tests", "e2e-qa-tests", "e2e-ui-tests", "e2e-dispatch"} {
		w := readWorkflow(t, "../../.github/workflows/"+file+".yaml")
		for name, job := range w.Jobs {
			if len(job.Steps) == 0 || name == "should-dispatch" {
				continue
			}
			t.Run(file+"/"+name, func(t *testing.T) {
				t.Parallel()
				gate, gateIndex := findStep(t, job, stepID("ci-decision"))
				assert.Equal(t, "bash", gate.Shell)
				findStep(t, job, actionRef("actions/checkout@"))
				for _, command := range commands[name] {
					step, _ := findStep(t, job, func(s workflowStep) bool { return strings.Contains(s.Run, command) })
					assert.NotContains(t, step.If, "always()", "ordinary tests must stop after setup or test failure")
				}
				if file != "e2e-dispatch" {
					assert.Contains(t, gate.Env["CI_SELECT_ACTION"], "github.event_name == 'workflow_dispatch' && 'run'")
					assert.Contains(t, gate.Env["CI_SELECT_ACTION"], "inputs.ci-action")
				}
				for index, step := range job.Steps {
					if step.ID == "ci-decision" {
						continue
					}
					assert.Less(t, gateIndex, index, "decide before checkout, setup, or validation")
					assert.Contains(t, step.If, "steps.ci-decision.outputs.action != 'skip'", step.Name)
					if step.ID == "junit2jira" {
						assert.Contains(t, step.If, "!cancelled()")
					}
					if strings.HasPrefix(step.Uses, "test-summary/action@") || strings.Contains(step.Name, "post-test") || step.Name == "Run final-post" || strings.HasPrefix(step.Name, "Teardown ") {
						assert.Contains(t, step.If, "always()", "retain failure/cancellation reporting semantics")
					}
				}
				for input, want := range map[string]string{"skip": "skip", "run": "run", "": "run", "SKIP": "run", "invalid": "run"} {
					output := filepath.Join(t.TempDir(), "output")
					cmd := exec.CommandContext(t.Context(), "bash", "-e", "-c", gate.Run)
					cmd.Env = append(os.Environ(), "GITHUB_OUTPUT="+output, "CI_SELECT_ACTION="+input)
					log, err := cmd.CombinedOutput()
					require.NoError(t, err, "%s", log)
					data, err := os.ReadFile(output)
					require.NoError(t, err)
					assert.Equal(t, "action="+want+"\n", string(data))
				}
			})
		}
	}
}

func TestE2EDispatchDecisions(t *testing.T) {
	t.Parallel()
	w := readWorkflow(t, "../../.github/workflows/e2e-dispatch.yaml")
	step, _ := findStep(t, w.Jobs["should-dispatch"], stepID("determine"))
	skip := make(map[string]string)
	for _, name := range e2eTargets() {
		skip[name] = "skip"
	}
	data, err := json.Marshal(skip)
	require.NoError(t, err)
	for name, c := range map[string]struct {
		decisions, result, event, fork, draft, uiOnly, labels string
		skipped, partial, allAdmitted                         bool
	}{
		"docs without optional labels": {decisions: string(data), skipped: true, allAdmitted: true},
		"docs on draft PR":             {decisions: string(data), draft: "true", skipped: true, allAdmitted: true},
		"partial selection":            {decisions: `{"e2e-byodb-tests":"skip"}`, partial: true},
		"planner failure":              {decisions: string(data), result: "failure"},
		"planner skipped":              {decisions: string(data), result: "skipped"},
		"empty map":                    {decisions: "{}"},
		"missing output":               {},
		"malformed JSON":               {decisions: "{"},
		"wrong result type":            {decisions: "[]"},
		"invalid value":                {decisions: `{"e2e-byodb-tests":"skip","other":false}`},
		"push ignores skips":           {decisions: string(data), event: "push", allAdmitted: true},
		"fork retains eligibility":     {decisions: string(data), fork: "true"},
		"UI retains eligibility":       {uiOnly: "true"},
		"optional label opts in":       {labels: "e2e-byodb-tests\n"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			gh := writeTestFile(t, filepath.Join(root, "gh"), "#!/usr/bin/env bash\nprintf '%s' \"$TEST_PR_LABELS\"\n")
			require.NoError(t, os.Chmod(gh, 0o755))
			output := filepath.Join(root, "output")
			event, result := c.event, c.result
			if event == "" {
				event = "pull_request"
			}
			if result == "" {
				result = "success"
			}
			cmd := exec.CommandContext(t.Context(), "python3", "-c", step.Run)
			cmd.Env = append(os.Environ(), "PATH="+root+string(os.PathListSeparator)+os.Getenv("PATH"),
				"GITHUB_OUTPUT="+output, "TEST_PR_LABELS="+c.labels, "input__event_name="+event,
				"input__ci_decisions="+c.decisions, "input__ci_planner_result="+result,
				"input__is_fork="+c.fork, "input__is_draft="+c.draft, "input__ui_only_change="+c.uiOnly,
				"input__repo=stackrox/stackrox", "input__pr_number=1")
			log, err := cmd.CombinedOutput()
			require.NoError(t, err, "%s", log)
			contents, err := os.ReadFile(output)
			require.NoError(t, err)
			outputs := make(map[string]string)
			for line := range strings.Lines(string(contents)) {
				key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
				require.True(t, ok)
				outputs[key] = value
			}
			var actions map[string]string
			var jobs map[string]bool
			require.NoError(t, json.Unmarshal([]byte(outputs["ci-actions"]), &actions))
			require.NoError(t, json.Unmarshal([]byte(outputs["jobs"]), &jobs))
			for _, target := range e2eTargets() {
				wantAction := "run"
				if c.skipped || (c.partial && target == "e2e-byodb-tests") {
					wantAction = "skip"
				}
				assert.Equal(t, wantAction, actions[target], target)
				if c.allAdmitted {
					assert.True(t, jobs[target], target)
				}
			}
			assert.Equal(t, c.skipped, outputs["skip-gha-images"] == "true")
			assert.Equal(t, c.skipped, outputs["skip-konflux-images"] == "true")
			if !c.allAdmitted {
				assert.Equal(t, c.fork != "true" && c.uiOnly != "true", jobs["e2e-gke-upgrade-tests"])
				assert.Equal(t, c.labels != "" || c.partial, jobs["e2e-byodb-tests"])
			}
		})
	}
}
