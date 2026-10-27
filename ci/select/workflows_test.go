package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

type ghaWorkflow struct {
	On   map[string]struct{ Types []string }
	Jobs map[string]struct {
		Name, If, Uses string
		Needs          yaml.Node
		Outputs        map[string]string
		Strategy       struct{ Matrix yaml.Node }
		Steps          []struct {
			ID, If, Name, Run, Uses string
			ContinueOnError         bool `yaml:"continue-on-error"`
		}
	}
}

func TestEnforcedJobsReachSelector(t *testing.T) {
	for file, names := range map[string][]string{
		"unit-tests.yaml":                   {"go-bench", "ui", "ui-component", "local-roxctl-tests", "shell-unit-tests", "sensor-integration-tests"},
		"style.yaml":                        {"check-generated-files", "golangci-lint", "check-dependent-images-exist", "github-actions-lint", "openshift-ci-lint"},
		"check-crd-compatibility.yaml":      {"check-crd-compatibility"},
		"scanner-db-integration-tests.yaml": {"db-integration-tests"},
	} {
		workflow := readWorkflow(t, file)
		for _, name := range names {
			t.Run(file+"/"+name, func(t *testing.T) {
				job := workflow.Jobs[name]
				assert.True(t, strings.HasPrefix(job.If,
					"contains(github.event.pull_request.labels.*.name, 'ci-dispatcher-enforce') ||"), job.If)
				assert.Contains(t, job.If, "needs.detect-changes.outputs.")
				foundGate := false
				for _, step := range job.Steps {
					foundGate = foundGate || step.Run == "ci/select/gha-gate.sh "+name
				}
				assert.True(t, foundGate)
			})
		}
	}
}

func TestCIValidationCannotBeSkipped(t *testing.T) {
	job := readWorkflow(t, "ci-select-tests.yaml").Jobs["selector-tests"]
	foundValidation := false
	for _, step := range job.Steps {
		if step.Uses == "./.github/actions/ci-select" {
			assert.False(t, step.ContinueOnError)
		}
		if step.Run == "ci/select/bin/ci-select validate" {
			assert.Empty(t, step.If)
			foundValidation = true
		}
		if step.ID == "ci-decision" {
			assert.True(t, foundValidation, "configuration validation must precede selection")
		}
	}
	assert.True(t, foundValidation)
}

func readWorkflow(t *testing.T, name string) ghaWorkflow {
	t.Helper()
	text, err := os.ReadFile(filepath.Join("../../.github/workflows", name))
	require.NoError(t, err)
	var workflow ghaWorkflow
	require.NoError(t, yaml.Unmarshal(text, &workflow))
	return workflow
}

func TestBuildSelectionPreservesRequiredChecks(t *testing.T) {
	workflow := readWorkflow(t, "build.yaml")
	matrixJob := workflow.Jobs["define-job-matrix"]
	assert.Contains(t, matrixJob.Outputs["build-action"], "ci-decision.outputs.action")
	for name, job := range workflow.Jobs {
		if name == "define-job-matrix" || name == "slack-on-build-failure" {
			continue
		}
		t.Run(name, func(t *testing.T) {
			assert.Empty(t, job.Name)
			assert.NotContains(t, job.If, "build-action")
			var needs []string
			if job.Needs.Kind == yaml.ScalarNode {
				needs = []string{job.Needs.Value}
			} else {
				require.NoError(t, job.Needs.Decode(&needs))
			}
			assert.Contains(t, needs, "define-job-matrix")
			require.NotEmpty(t, job.Steps)
			for _, step := range job.Steps {
				assert.Contains(t, step.If, "needs.define-job-matrix.outputs.build-action != 'skip'", step.Name+step.Uses)
			}
		})
	}
	assert.Contains(t, workflow.On["pull_request"].Types, "labeled")
	assert.Contains(t, workflow.On["pull_request"].Types, "unlabeled")
}

func TestRequiredJobWorkGates(t *testing.T) {
	for file, name := range map[string]string{
		"unit-tests.yaml": "go-bench", "style.yaml": "check-dependent-images-exist",
	} {
		job := readWorkflow(t, file).Jobs[name]
		assert.NotContains(t, job.If, "ci-decision")
		foundGate := false
		for _, step := range job.Steps {
			if step.ID == "ci-decision" {
				foundGate = true
				assert.Equal(t, "ci/select/gha-gate.sh "+name, step.Run)
				continue
			}
			if foundGate {
				assert.Contains(t, step.If, "steps.ci-decision.outputs.action != 'skip'", step.Name+step.Uses)
			}
		}
		assert.True(t, foundGate, name)
	}
}

func TestImageWaitSelectionPreservesRequiredChecks(t *testing.T) {
	workflow := readWorkflow(t, "e2e-dispatch.yaml")
	for name, output := range map[string]string{
		"wait-for-images": "wait-for-gha-images", "wait-for-konflux-images": "wait-for-konflux-images",
	} {
		job := workflow.Jobs[name]
		assert.Empty(t, job.If)
		require.NotEmpty(t, job.Steps)
		for _, step := range job.Steps {
			assert.Contains(t, step.If, "needs.should-dispatch.outputs."+output+" == 'true'")
		}
	}
	for _, job := range dispatchJobs {
		assert.Empty(t, workflow.Jobs[job.name].If, job.name)
		assert.NotEmpty(t, workflow.Jobs[job.name].Uses, job.name)
	}
}

func TestRequiredGHACheckInventory(t *testing.T) {
	text, err := os.ReadFile("testdata/gha-required-checks.json")
	require.NoError(t, err)
	var required map[string]map[string][]string
	require.NoError(t, json.Unmarshal(text, &required))
	build := readWorkflow(t, "build.yaml")
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "scripts/ci/lib.sh"), `is_in_PR_context() { return 0; }
is_tagged() { return 1; }
pr_has_label() { return 1; }
`, 0o600)
	outputPath := filepath.Join(root, "output")
	var matrixScript string
	for _, step := range build.Jobs["define-job-matrix"].Steps {
		if step.ID == "define-job-matrix" {
			assert.Empty(t, step.If)
			matrixScript = step.Run
		}
	}
	require.NotEmpty(t, matrixScript)
	command := exec.CommandContext(t.Context(), "bash", "-c", matrixScript)
	command.Dir = root
	command.Env = append(command.Environ(), "GITHUB_OUTPUT="+outputPath)
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	text, err = os.ReadFile(outputPath)
	require.NoError(t, err)
	var matrices map[string]map[string][]string
	require.NoError(t, json.Unmarshal([]byte(strings.TrimPrefix(strings.TrimSpace(string(text)), "matrix=")), &matrices))
	matrixKey := regexp.MustCompile(`\)\.([a-z_]+)`)
	count := 0
	for file, jobs := range required {
		workflow := readWorkflow(t, file)
		for id, contexts := range jobs {
			job, exists := workflow.Jobs[id]
			require.True(t, exists, file+": "+id)
			assert.Empty(t, job.Name, id)
			for _, context := range contexts {
				count++
				if prefix, child, nested := strings.Cut(context, " / "); nested {
					assert.Equal(t, id, prefix)
					childWorkflow := readWorkflow(t, filepath.Base(job.Uses))
					assert.Contains(t, childWorkflow.Jobs, child)
					continue
				}
				if file != "build.yaml" {
					assert.True(t, context == id || strings.HasPrefix(context, id+" ("), context)
					continue
				}
				key := matrixKey.FindStringSubmatch(job.Strategy.Matrix.Value)
				require.Len(t, key, 2, id)
				coordinates := strings.TrimSuffix(strings.TrimPrefix(context, id+" ("), ")")
				for coordinate := range strings.SplitSeq(coordinates, ", ") {
					found := false
					for _, values := range matrices[key[1]] {
						found = found || slices.Contains(values, coordinate)
					}
					assert.True(t, found, context+": "+coordinate)
				}
			}
		}
	}
	assert.Equal(t, 22, count)
}
