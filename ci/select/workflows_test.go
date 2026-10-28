package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

type (
	workflow struct {
		On map[string]struct {
			Outputs map[string]struct {
				Value string `yaml:"value"`
			} `yaml:"outputs"`
		} `yaml:"on"`
		Jobs        map[string]workflowJob `yaml:"jobs"`
		Concurrency struct {
			Group            string `yaml:"group"`
			CancelInProgress bool   `yaml:"cancel-in-progress"`
		} `yaml:"concurrency"`
	}
	workflowJob struct {
		If       string `yaml:"if"`
		Needs    any    `yaml:"needs"`
		Uses     string `yaml:"uses"`
		Strategy struct {
			FailFast bool           `yaml:"fail-fast"`
			Matrix   map[string]any `yaml:"matrix"`
		} `yaml:"strategy"`
		Steps   []workflowStep    `yaml:"steps"`
		Outputs map[string]string `yaml:"outputs"`
	}
	workflowStep struct {
		ID    string            `yaml:"id"`
		If    string            `yaml:"if"`
		Uses  string            `yaml:"uses"`
		Run   string            `yaml:"run"`
		Shell string            `yaml:"shell"`
		With  map[string]string `yaml:"with"`
		Env   map[string]string `yaml:"env"`
	}
	expressionCase struct {
		Name       string         `json:"name"`
		Expression string         `json:"expression"`
		Context    map[string]any `json:"context"`
		Status     string         `json:"status"`
		Want       any            `json:"want"`
	}
)

func readWorkflow(t *testing.T, path string) workflow {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var w workflow
	require.NoError(t, yaml.Unmarshal(data, &w))
	return w
}

func findStep(t *testing.T, job workflowJob, predicate func(workflowStep) bool) (workflowStep, int) {
	t.Helper()
	index := slices.IndexFunc(job.Steps, predicate)
	require.NotEqual(t, -1, index, "required step missing")
	return job.Steps[index], index
}

func stepID(id string) func(workflowStep) bool {
	return func(s workflowStep) bool { return s.ID == id }
}

func actionRef(ref string) func(workflowStep) bool {
	return func(s workflowStep) bool { return strings.HasPrefix(s.Uses, ref) }
}

func testExpressions(t *testing.T, cases []expressionCase) {
	t.Helper()
	if _, err := os.Stat("node_modules/@actions/expressions"); os.IsNotExist(err) {
		if os.Getenv("CI_SELECT_EXPRESSION_TESTS") == "required" {
			t.Fatal("install expression tests with npm ci --prefix ci/select before running Go tests")
		}
		t.Skip("expression scenarios require npm ci --prefix ci/select")
	}
	input, err := json.Marshal(cases)
	require.NoError(t, err)
	cmd := exec.CommandContext(t.Context(), "node", "workflow-expressions.mjs")
	cmd.Stdin = bytes.NewReader(input)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
}

func workflowContext(event, action, label string, enforce bool, planner string) map[string]any {
	labels := []map[string]string{}
	if enforce {
		labels = append(labels, map[string]string{"name": enforceLabel})
	}
	return map[string]any{
		"github": map[string]any{
			"event_name": event, "head_ref": "test-branch", "run_id": 123,
			"event": map[string]any{
				"action": action, "label": map[string]string{"name": label},
				"pull_request": map[string]any{"labels": labels},
			},
		},
		"needs": map[string]any{
			"ci-selection":   map[string]any{"result": planner},
			"detect-changes": map[string]any{"outputs": map[string]string{"ui_only": ""}},
		},
		"matrix": map[string]string{"pg": "16"},
	}
}

func TestWorkflowLabelEvents(t *testing.T) {
	t.Parallel()
	w := readWorkflow(t, "../../.github/workflows/unit-tests.yaml")
	assert.True(t, w.Concurrency.CancelInProgress)
	var cases []expressionCase
	for name, c := range map[string]struct {
		event, action, label, planner, status string
		enforce, uiOnly, admit                bool
	}{
		"PR update":                 {event: "pull_request", action: "synchronize", planner: "success", status: "success", admit: true},
		"planner failure":           {event: "pull_request", action: "synchronize", planner: "failure", status: "failure", uiOnly: true, admit: true},
		"missing planner":           {event: "pull_request", action: "synchronize", planner: "skipped", status: "success", admit: true},
		"cancelled":                 {event: "pull_request", action: "synchronize", planner: "cancelled", status: "cancelled"},
		"preview UI filter":         {event: "pull_request", action: "synchronize", planner: "success", status: "success", uiOnly: true, admit: true},
		"enforced UI selection":     {event: "pull_request", action: "synchronize", planner: "success", status: "success", uiOnly: true, enforce: true, admit: true},
		"enable enforcement":        {event: "pull_request", action: "labeled", label: enforceLabel, planner: "success", status: "success", enforce: true, admit: true},
		"disable enforcement":       {event: "pull_request", action: "unlabeled", label: enforceLabel, planner: "success", status: "success", admit: true},
		"unrelated label":           {event: "pull_request", action: "labeled", label: "area/ci", planner: "skipped", status: "success", enforce: true},
		"release build label event": {event: "pull_request", action: "labeled", label: "ci-release-build", planner: "skipped", status: "success", enforce: true},
		"unrelated unlabel":         {event: "pull_request", action: "unlabeled", label: "area/ci", planner: "failure", status: "failure"},
		"push":                      {event: "push", planner: "success", status: "success", admit: true},
	} {
		context := workflowContext(c.event, c.action, c.label, c.enforce, c.planner)
		if c.uiOnly {
			context["needs"].(map[string]any)["detect-changes"] = map[string]any{"outputs": map[string]string{"ui_only": "true"}}
		}
		for _, jobName := range []string{"go", "go-postgres", "go-bench", "sensor-integration-tests"} {
			want := c.admit
			if (jobName == "go-bench" || jobName == "sensor-integration-tests") && c.uiOnly && !c.enforce && c.planner == "success" {
				want = false
			}
			job, ok := w.Jobs[jobName]
			require.True(t, ok)
			cases = append(cases, expressionCase{name + "/" + jobName, job.If, context, c.status, want})
		}
		for _, jobName := range []string{"ci-selection", "detect-changes"} {
			// These jobs have no failing dependencies; cancellation supplies the status.
			status := "success"
			if c.status == "cancelled" {
				status = "cancelled"
			}
			cases = append(cases, expressionCase{name + "/" + jobName, w.Jobs[jobName].If, context, status, c.admit})
		}
		group := strings.TrimPrefix(strings.TrimSpace(w.Concurrency.Group), "${{ github.workflow }}-")
		wantGroup := "test-branch"
		if (c.action == "labeled" || c.action == "unlabeled") && c.label != enforceLabel {
			wantGroup = "ignored-label-123"
		}
		cases = append(cases, expressionCase{name + "/concurrency", group, context, "success", wantGroup})
	}
	testExpressions(t, cases)
}

func TestWorkflowGates(t *testing.T) {
	t.Parallel()
	w := readWorkflow(t, "../../.github/workflows/unit-tests.yaml")
	for name, command := range map[string]string{
		"go": "make go-unit-tests", "go-postgres": "make go-postgres-unit-tests",
		"go-bench": "make go-postgres-bench-tests", "sensor-integration-tests": "make sensor-integration-test",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			job, ok := w.Jobs[name]
			require.True(t, ok)
			needs, err := json.Marshal(job.Needs)
			require.NoError(t, err)
			assert.Contains(t, string(needs), `"ci-selection"`)
			gate, gateIndex := findStep(t, job, stepID("ci-decision"))
			assert.Contains(t, gate.Run, "gha-consume.sh "+name)
			assert.Contains(t, gate.Env["CI_SELECT_DECISIONS"], "needs.ci-selection.outputs.decisions")
			assert.Contains(t, gate.Env["CI_SELECT_PLANNER_RESULT"], "needs.ci-selection.result")
			checkout, checkoutIndex := findStep(t, job, actionRef("actions/checkout@"))
			assert.NotEmpty(t, checkout.Uses)
			assert.Less(t, checkoutIndex, gateIndex)
			preamble, preambleIndex := findStep(t, job, actionRef("./.github/actions/job-preamble"))
			assert.Less(t, gateIndex, preambleIndex)
			assert.Equal(t, "${{ secrets.GCP_SERVICE_ACCOUNT_STACKROX_CI }}", preamble.With["gcp-account"])
			if name == "go" {
				assert.Equal(t, "30", preamble.With["free-disk-space"])
			}
			_, testIndex := findStep(t, job, func(s workflowStep) bool { return strings.Contains(s.Run, command) })
			assert.Less(t, preambleIndex, testIndex)
			findStep(t, job, actionRef("./.github/actions/cache-go-dependencies"))
			findStep(t, job, stepID("junit-report-1"))
			findStep(t, job, actionRef("test-summary/action@"))
			if name == "go-bench" {
				findStep(t, job, stepID("postgres-diagnostics"))
			} else {
				jira, jiraIndex := findStep(t, job, stepID("junit2jira"))
				assert.Equal(t, "./.github/actions/junit2jira", jira.Uses)
				assert.Less(t, testIndex, jiraIndex)
				assert.Equal(t, "${{ steps.junit2jira.outputs.new-jiras }}", job.Outputs["new-jiras"])
			}
			if name == "go" {
				for id, cmd := range map[string]string{"integration-tests": "make integration-unit-tests", "operator-tests": "make -C operator/ test-integration", "helm-tests": "make -C operator/ test-helm"} {
					step, index := findStep(t, job, stepID(id))
					assert.Contains(t, step.Run, cmd)
					assert.Less(t, testIndex, index)
				}
				findStep(t, job, stepID("junit-report-2"))
				findStep(t, job, stepID("publish-report-2"))
			}
			if name == "go" || name == "go-postgres" {
				assert.False(t, job.Strategy.FailFast)
				assert.ElementsMatch(t, []any{`GOTAGS=""`, "GOTAGS=release"}, job.Strategy.Matrix["gotags"])
				require.NotEmpty(t, job.Strategy.Matrix["exclude"])
			}
			var cases []expressionCase
			if name == "go" || name == "go-postgres" {
				exclusions, ok := job.Strategy.Matrix["exclude"].([]any)
				require.True(t, ok)
				var releaseExclusion string
				for _, value := range exclusions {
					exclude, ok := value.(map[string]any)
					require.True(t, ok)
					if expression, ok := exclude["gotags"].(string); ok && strings.Contains(expression, "ci-release-build") {
						releaseExclusion = expression
					}
				}
				require.NotEmpty(t, releaseExclusion, "release variant must remain opt-in on PRs")
				for _, release := range []bool{false, true} {
					context := workflowContext("pull_request", "synchronize", "", false, "success")
					labels := []map[string]string{}
					if release {
						labels = append(labels, map[string]string{"name": "ci-release-build"})
					}
					context["github"].(map[string]any)["event"].(map[string]any)["pull_request"] = map[string]any{"labels": labels}
					var want any = "GOTAGS=release"
					if release {
						want = false
					}
					cases = append(cases, expressionCase{"release matrix variant", releaseExclusion, context, "success", want})
				}
			}
			for index, step := range job.Steps {
				assert.NotEqual(t, "./.github/actions/ci-select", step.Uses)
				relevant := index == testIndex ||
					slices.Contains([]string{"postgres-host", "postgres-container", "postgres-ready", "postgres-diagnostics", "integration-tests", "operator-tests", "helm-tests", "load-images", "junit2jira"}, step.ID) ||
					strings.HasPrefix(step.ID, "junit-report-") || strings.HasPrefix(step.ID, "publish-report-") ||
					strings.Contains(step.Uses, "cache-go-dependencies") || strings.Contains(step.Uses, "job-preamble") ||
					strings.Contains(step.Uses, "codecov/") || strings.Contains(step.Uses, "docker/login-action") || strings.Contains(step.Uses, "helm/kind-action")
				if !relevant {
					continue
				}
				for phase, status := range map[string]string{"selected": "success", "setup failure": "failure", "test failure": "failure", "cancelled": "cancelled"} {
					for _, selection := range []string{"run", "skip", ""} {
						for _, pg := range []string{"16", "default"} {
							context := workflowContext("pull_request", "synchronize", "", true, "success")
							context["steps"] = map[string]any{"ci-decision": map[string]any{"outputs": map[string]string{"action": selection}}}
							context["matrix"] = map[string]string{"pg": pg}
							want := selection != "skip"
							switch {
							case strings.HasPrefix(step.ID, "junit-report-"), strings.HasPrefix(step.ID, "publish-report-"):
							case step.ID == "junit2jira":
								want = want && status != "cancelled"
							case step.ID == "postgres-diagnostics":
								want = want && status == "failure"
							default:
								want = want && status == "success"
								if name == "go-postgres" {
									if step.ID == "postgres-host" {
										want = want && pg == "default"
									}
									if step.ID == "postgres-container" {
										want = want && pg != "default"
									}
								}
							}
							cases = append(cases, expressionCase{phase + "/" + selection + "/" + pg + "/" + step.ID + step.Uses, step.If, context, status, want})
						}
					}
				}
				if strings.HasPrefix(step.ID, "junit-report-") {
					assert.Contains(t, step.Run, "make generate-junit-reports")
				}
				if step.Run != "" {
					assert.Equal(t, "bash -e {0}", step.Shell)
				}
			}
			testExpressions(t, cases)
		})
	}
}

func TestReusablePlanner(t *testing.T) {
	t.Parallel()
	caller := readWorkflow(t, "../../.github/workflows/unit-tests.yaml")
	calls := 0
	for _, job := range caller.Jobs {
		if job.Uses == "./.github/workflows/ci-selection.yaml" {
			calls++
			assert.Empty(t, job.Strategy.Matrix, "plan outside the test matrices")
		}
	}
	assert.Equal(t, 1, calls, "compute once per workflow run")
	planner := readWorkflow(t, "../../.github/workflows/ci-selection.yaml")
	output := planner.On["workflow_call"].Outputs["decisions"].Value
	assert.Equal(t, "${{ jobs.plan.outputs.decisions }}", output)
	plan, ok := planner.Jobs["plan"]
	require.True(t, ok)
	assert.Equal(t, "${{ steps.selection.outputs.decisions }}", plan.Outputs["decisions"])
	checkout, checkoutIndex := findStep(t, plan, actionRef("actions/checkout@"))
	assert.Equal(t, "0", checkout.With["fetch-depth"])
	selectStep, selectIndex := findStep(t, plan, stepID("selection"))
	assert.Less(t, checkoutIndex, selectIndex)
	assert.Equal(t, "./.github/actions/ci-select", selectStep.Uses)
	validation := readWorkflow(t, "../../.github/workflows/ci-select-tests.yaml").Jobs["selector-tests"]
	assert.Empty(t, validation.If)
	for _, command := range []string{"ci-select validate", "go test -race ./ci/select", "npm ci --prefix ci/select"} {
		step, _ := findStep(t, validation, func(s workflowStep) bool { return strings.Contains(s.Run, command) })
		assert.Empty(t, step.If, "selection must not suppress its validation")
		if strings.Contains(command, "go test") {
			assert.Equal(t, "required", step.Env["CI_SELECT_EXPRESSION_TESTS"])
		}
	}
}
