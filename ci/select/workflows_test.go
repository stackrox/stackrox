package main

import (
	"encoding/json"
	"os"
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
			Types  []string `yaml:"types"`
			Inputs map[string]struct {
				Default string `yaml:"default"`
			} `yaml:"inputs"`
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
		If       string            `yaml:"if"`
		Needs    any               `yaml:"needs"`
		Uses     string            `yaml:"uses"`
		With     map[string]string `yaml:"with"`
		Strategy struct {
			FailFast bool           `yaml:"fail-fast"`
			Matrix   map[string]any `yaml:"matrix"`
		} `yaml:"strategy"`
		Steps   []workflowStep    `yaml:"steps"`
		Outputs map[string]string `yaml:"outputs"`
	}
	workflowStep struct {
		Name  string            `yaml:"name"`
		ID    string            `yaml:"id"`
		If    string            `yaml:"if"`
		Uses  string            `yaml:"uses"`
		Run   string            `yaml:"run"`
		Shell string            `yaml:"shell"`
		With  map[string]string `yaml:"with"`
		Env   map[string]string `yaml:"env"`
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

func TestWorkflowLabelEvents(t *testing.T) {
	t.Parallel()
	w := readWorkflow(t, "../../.github/workflows/unit-tests.yaml")
	for _, name := range []string{"ci-selection", "detect-changes", "go", "go-postgres", "go-bench", "sensor-integration-tests"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			job, ok := w.Jobs[name]
			require.True(t, ok)
			for _, guard := range []string{
				"github.event_name != 'pull_request'",
				"github.event.action != 'labeled'",
				"github.event.action != 'unlabeled'",
				"github.event.label.name == 'ci-dispatcher-enforce'",
			} {
				assert.Contains(t, job.If, guard, "unrelated label events must not start test work")
			}
		})
	}
	assert.True(t, w.Concurrency.CancelInProgress)
	assert.Contains(t, w.Concurrency.Group, "github.event.label.name != 'ci-dispatcher-enforce'")
	assert.Contains(t, w.Concurrency.Group, "ignored-label-")
	assert.Contains(t, w.Concurrency.Group, "github.run_id", "ignored events need their own concurrency group")
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
			assert.Contains(t, job.If, "!cancelled()", "planner failure must not suppress dependent jobs")
			if name == "go-bench" || name == "sensor-integration-tests" {
				assert.Contains(t, job.If, "needs.ci-selection.result != 'success'")
				assert.Contains(t, job.If, "ci-dispatcher-enforce")
				assert.Contains(t, job.If, "!needs.detect-changes.outputs.ui_only")
			}
			if name == "go" || name == "go-postgres" {
				exclusions, ok := job.Strategy.Matrix["exclude"].([]any)
				require.True(t, ok)
				assert.True(t, slices.ContainsFunc(exclusions, func(value any) bool {
					exclude, ok := value.(map[string]any)
					if !ok {
						return false
					}
					expression, ok := exclude["gotags"].(string)
					return ok && strings.Contains(expression, "ci-release-build") &&
						strings.Contains(expression, "github.event_name == 'pull_request'")
				}), "release variant must remain opt-in on PRs")
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
				assert.Contains(t, step.If, "steps.ci-decision.outputs.action != 'skip'", step.ID+step.Uses)
				switch {
				case strings.HasPrefix(step.ID, "junit-report-"), strings.HasPrefix(step.ID, "publish-report-"):
					assert.Contains(t, step.If, "always()")
				case step.ID == "junit2jira":
					assert.Contains(t, step.If, "!cancelled()")
				case step.ID == "postgres-diagnostics":
					assert.Contains(t, step.If, "failure()")
					assert.Contains(t, step.If, "!cancelled()")
				default:
					for _, status := range []string{"always()", "failure()", "cancelled()"} {
						assert.NotContains(t, step.If, status, "ordinary steps must require success")
					}
				}
				if strings.HasPrefix(step.ID, "junit-report-") {
					assert.Contains(t, step.Run, "make generate-junit-reports")
				}
				if step.Run != "" {
					assert.Equal(t, "bash -e {0}", step.Shell)
				}
			}
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
	for _, command := range []string{"ci-select validate", "go test -race ./ci/select"} {
		step, _ := findStep(t, validation, func(s workflowStep) bool { return strings.Contains(s.Run, command) })
		assert.Empty(t, step.If, "selection must not suppress its validation")
	}
}
