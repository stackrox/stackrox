package main

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

type (
	workflow struct {
		Jobs        map[string]workflowJob `yaml:"jobs"`
		Concurrency struct {
			Group            string `yaml:"group"`
			CancelInProgress bool   `yaml:"cancel-in-progress"`
		} `yaml:"concurrency"`
	}
	workflowJob struct {
		If       string            `yaml:"if"`
		Needs    any               `yaml:"needs"`
		RunsOn   string            `yaml:"runs-on"`
		Strategy any               `yaml:"strategy"`
		Steps    []workflowStep    `yaml:"steps"`
		Outputs  map[string]string `yaml:"outputs"`
	}
	workflowStep struct {
		Name  string            `yaml:"name"`
		ID    string            `yaml:"id"`
		If    string            `yaml:"if"`
		Uses  string            `yaml:"uses"`
		Run   string            `yaml:"run"`
		Shell string            `yaml:"shell"`
		With  map[string]string `yaml:"with"`
	}
	compositeAction struct {
		Inputs map[string]struct {
			Required bool `yaml:"required"`
		} `yaml:"inputs"`
		Runs struct {
			Using string         `yaml:"using"`
			Steps []workflowStep `yaml:"steps"`
		} `yaml:"runs"`
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

func TestWorkflowLabelEvents(t *testing.T) {
	t.Parallel()
	w := readWorkflow(t, "../../.github/workflows/unit-tests.yaml")
	assert.Equal(t, "github.event_name != 'pull_request' || "+
		"(github.event.action != 'labeled' && github.event.action != 'unlabeled') || "+
		"github.event.label.name == 'ci-dispatcher-enforce'",
		strings.Join(strings.Fields(w.Jobs["detect-changes"].If), " "))
	assert.Equal(t, "${{ github.workflow }}-${{ github.event_name == 'pull_request' && "+
		"(github.event.action == 'labeled' || github.event.action == 'unlabeled') && "+
		"github.event.label.name != 'ci-dispatcher-enforce' && "+
		"format('ignored-label-{0}', github.run_id) || github.head_ref || github.run_id }}",
		strings.Join(strings.Fields(w.Concurrency.Group), " "))
	assert.True(t, w.Concurrency.CancelInProgress)
	for name, job := range w.Jobs {
		if name != "detect-changes" && name != "slack-on-unit-tests-failure" {
			assert.Equal(t, "detect-changes", job.Needs, name)
		}
	}
}

func TestWorkflowGates(t *testing.T) {
	t.Parallel()
	w := readWorkflow(t, "../../.github/workflows/unit-tests.yaml")
	for name, action := range map[string]string{
		"go": "go-suite", "go-postgres": "go-postgres-suite",
		"go-bench": "go-bench-suite", "sensor-integration-tests": "sensor-integration-suite",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			job := w.Jobs[name]
			assert.Equal(t, "ubuntu-24.04", job.RunsOn)
			require.Greater(t, len(job.Steps), 2)
			assert.Contains(t, job.Steps[0].Uses, "actions/checkout@")
			assert.Equal(t, "${{ github.event_name == 'pull_request' && '0' || '1' }}", job.Steps[0].With["fetch-depth"])
			gate := job.Steps[1]
			assert.Equal(t, "ci-decision", gate.ID)
			assert.Equal(t, "github.event_name == 'pull_request'", gate.If)
			assert.Equal(t, "./.github/actions/ci-select", gate.Uses)
			assert.Equal(t, name, gate.With["job"])
			suite := job.Steps[2]
			assert.Equal(t, "./.github/actions/"+action, suite.Uses)
			assert.Equal(t, "always() && steps.ci-decision.outputs.action != 'skip'", suite.If)
			data, err := os.ReadFile("../../.github/actions/" + action + "/action.yaml")
			require.NoError(t, err)
			var composite compositeAction
			require.NoError(t, yaml.Unmarshal(data, &composite))
			assert.Equal(t, "composite", composite.Runs.Using)
			require.NotEmpty(t, composite.Runs.Steps)
			require.Len(t, suite.With, len(composite.Inputs))
			for input, definition := range composite.Inputs {
				assert.True(t, definition.Required, input)
				assert.NotEmpty(t, suite.With[input], input)
			}
			for _, step := range composite.Runs.Steps {
				assert.NotContains(t, step.If, "ci-decision")
				switch {
				case step.Name == "Postgres diagnostics":
					assert.Equal(t, "(!cancelled()) && (failure() || job.status == 'failure')", step.If)
				case strings.Contains(step.Name, "junit report"), step.Name == "Publish Test Report":
					assert.Equal(t, "always()", step.If)
				default:
					assert.Contains(t, step.If, "job.status == 'success'", step.Name+step.Uses)
				}
				if step.Run != "" {
					assert.Equal(t, "bash -e {0}", step.Shell, step.Name)
				}
			}
			if name == "go-bench" {
				assert.Len(t, job.Steps, 3)
			} else {
				require.Len(t, job.Steps, 4)
				jira := job.Steps[3]
				assert.Equal(t, "junit2jira", jira.ID)
				assert.Equal(t, "./.github/actions/junit2jira", jira.Uses)
				assert.Equal(t, "(!cancelled()) && steps.ci-decision.outputs.action != 'skip'", jira.If)
				assert.Equal(t, "${{ steps.junit2jira.outputs.new-jiras }}", job.Outputs["new-jiras"])
			}
			if name == "go-bench" || name == "sensor-integration-tests" {
				assert.Contains(t, job.If, "github.event_name == 'pull_request'")
				assert.Contains(t, job.If, "ci-dispatcher-enforce")
				assert.Contains(t, job.If, "!needs.detect-changes.outputs.ui_only")
			} else {
				assert.Empty(t, job.If)
				require.NotNil(t, job.Strategy)
			}
		})
	}
	for _, name := range []string{"ui", "ui-component", "local-roxctl-tests", "shell-unit-tests", "openshift-ci-unit-tests"} {
		for _, step := range w.Jobs[name].Steps {
			assert.NotEqual(t, "./.github/actions/ci-select", step.Uses)
		}
		assert.NotContains(t, w.Jobs[name].If, "ci-dispatcher-enforce")
	}
	validation := readWorkflow(t, "../../.github/workflows/ci-select-tests.yaml").Jobs["selector-tests"]
	assert.Empty(t, validation.If)
	for _, step := range validation.Steps {
		if strings.Contains(step.Run, " validate") || strings.Contains(step.Run, "go test") {
			assert.Empty(t, step.If, "selection must not suppress its validation")
		}
	}
}
