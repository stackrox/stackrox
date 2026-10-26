package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

type yamlObject = map[string]any

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	require.NoError(t, err)
	return root
}

func workflow(t *testing.T, name string) yamlObject {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repositoryRoot(t), ".github/workflows", name))
	require.NoError(t, err)
	var result yamlObject
	require.NoError(t, yaml.Unmarshal(data, &result))
	return result
}

func object(value any) yamlObject { return value.(map[string]any) }

func bash(t *testing.T) string {
	t.Helper()
	if override := os.Getenv("NATIVE_TEST_BASH"); override != "" {
		return override
	}
	if runtime.GOOS == "darwin" {
		return "/opt/homebrew/bin/bash"
	}
	return "bash"
}

func runShell(t *testing.T, script, dir string, env []string) (string, error) {
	t.Helper()
	cmd := exec.Command(bash(t), "-c", script)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func TestActualCollector(t *testing.T) {
	w := workflow(t, "detect-changes.yml")
	steps := object(object(w["jobs"])["detect"])["steps"].([]any)
	script := object(steps[0])["run"].(string)
	for name, test := range map[string]struct {
		pages             string
		count             int
		stale, apiFailure bool
		wantPaths         []string
		complete          bool
	}{
		"should read pages":                         {pages: `[[{"filename":"README.md"}],[{"filename":"CHANGELOG.md"}]]`, count: 2, wantPaths: []string{"README.md", "CHANGELOG.md"}, complete: true},
		"should retain old rename paths":            {pages: `[[{"filename":"README.md","previous_filename":"product.go"}]]`, count: 1, wantPaths: []string{"README.md", "product.go"}, complete: true},
		"should preserve empty diff":                {pages: `[[]]`, count: 0, wantPaths: []string{}, complete: true},
		"should reject truncated data":              {pages: `[[{"filename":"README.md"}]]`, count: 2},
		"should reject malformed pages":             {pages: `[{"filename":"README.md"}]`, count: 1},
		"should reject newline":                     {pages: `[[{"filename":"a\nb.md"}]]`, count: 1},
		"should reject stale revisions":             {pages: `[[{"filename":"README.md"}]]`, count: 1, stale: true},
		"should retain execution after API failure": {pages: `[[]]`, count: 0, apiFailure: true},
		"should reject API limit":                   {pages: `[[]]`, count: 3000},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			out := filepath.Join(dir, "output")
			meta := fmt.Sprintf(`{"base":{"sha":%q},"head":{"sha":%q},"changed_files":%d}`, strings.Repeat("a", 40), strings.Repeat("b", 40), test.count)
			if test.stale {
				meta = strings.Replace(meta, strings.Repeat("b", 40), strings.Repeat("c", 40), 1)
			}
			fake := `#!/usr/bin/env bash
if [[ "${API_FAIL:-}" == true ]]; then exit 1; fi
if [[ "$*" == *'/files'* ]]; then printf '%s\n' "$PAGES"; else printf '%s\n' "$META"; fi
`
			require.NoError(t, os.WriteFile(filepath.Join(dir, "gh"), []byte(fake), 0o700))
			env := []string{"PATH=" + dir + ":" + os.Getenv("PATH"), "PAGES=" + test.pages, "META=" + meta, fmt.Sprintf("API_FAIL=%t", test.apiFailure), "PR_NUMBER=1", "GH_REPO=stackrox/stackrox", "EXPECTED_BASE=" + strings.Repeat("a", 40), "EXPECTED_HEAD=" + strings.Repeat("b", 40), "RUN_ALL=", "WORKFLOW_FILE_PATH=.github/workflows/detect-changes.yml", "RUNNER_TEMP=" + dir, "GITHUB_OUTPUT=" + out}
			text, err := runShell(t, script, repositoryRoot(t), env)
			require.NoError(t, err, text)
			payload, err := os.ReadFile(filepath.Join(dir, "ci-selection-files.json"))
			if !test.complete {
				require.ErrorIs(t, err, os.ErrNotExist)
				return
			}
			require.NoError(t, err)
			var diff changedFiles
			require.NoError(t, json.Unmarshal(payload, &diff))
			require.ElementsMatch(t, test.wantPaths, diff.Files)
			facts, err := classify(diff, map[string]pathGroup{"docs": {Include: []string{`^(README\.md|CHANGELOG\.md)$`}}}, nil)
			require.NoError(t, err)
			if slices.Contains(test.wantPaths, "product.go") {
				require.Empty(t, facts.Every)
			}
		})
	}
	job := object(object(w["jobs"])["detect"])
	require.Contains(t, object(job["outputs"])["facts"], "steps.classify.outputs.facts")
	require.Contains(t, object(object(object(w["on"])["workflow_call"])["outputs"])["facts"].(map[string]any)["value"], "jobs.detect.outputs.facts")
}

func TestProductStepGuards(t *testing.T) {
	jobs := map[string][]string{
		"unit-tests.yaml":                   {"go", "go-postgres", "go-bench", "ui", "ui-component", "local-roxctl-tests", "shell-unit-tests", "sensor-integration-tests"},
		"scanner-db-integration-tests.yaml": {"db-integration-tests"},
		"check-crd-compatibility.yaml":      {"check-crd-compatibility"},
		"scanner-e2e-test.yaml":             {"scanner-ft-images", "scanner-ft-run"},
		"performance-tests.yml":             {"wait-for-images", "run-k6-performance-tests"},
	}
	for name, targets := range jobs {
		t.Run(name, func(t *testing.T) {
			w := workflow(t, name)
			all := object(w["jobs"])
			require.Equal(t, true, object(object(all["detect-changes"])["with"])["classify"])
			for _, target := range targets {
				job := object(all[target])
				require.Contains(t, object(job["env"])["CI_DOCS_ONLY"], ".complete == true")
				for _, value := range job["steps"].([]any) {
					step := object(value)
					if step["name"] == "Explain docs-only selection" {
						continue
					}
					require.Contains(t, step["if"], "env.CI_DOCS_ONLY != 'true'", target)
				}
			}
		})
	}
	style := workflow(t, "style.yaml")
	require.NotContains(t, object(style["jobs"])["detect-changes"], "with")
}

func TestActualE2EDispatch(t *testing.T) {
	w := workflow(t, "e2e-dispatch.yaml")
	job := object(object(w["jobs"])["should-dispatch"])
	script := object(job["steps"].([]any)[0])["run"].(string)
	for name, test := range map[string]struct {
		facts       string
		draft, fork bool
		labels      string
		wantAny     bool
	}{
		"should skip docs even when explicitly requested": {facts: `{"version":1,"complete":true,"every":["docs"]}`, labels: "e2e-qa-tests-gke", wantAny: false},
		"should retain execution when unavailable":        {facts: `{"version":1,"complete":false,"every":[]}`, wantAny: true},
		"should retain draft restrictions":                {facts: `{}`, draft: true, wantAny: false},
		"should retain fork restrictions":                 {facts: `{}`, fork: true, wantAny: false},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			out := filepath.Join(dir, "output")
			require.NoError(t, os.WriteFile(filepath.Join(dir, "gh"), []byte("#!/usr/bin/env bash\nprintf '%s\\n' \"$LABELS\"\n"), 0o700))
			cmd := exec.Command("python3", "-c", script)
			cmd.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"), "LABELS="+test.labels, "input__file_facts="+test.facts, "input__event_name=pull_request", fmt.Sprintf("input__is_draft=%t", test.draft), fmt.Sprintf("input__is_fork=%t", test.fork), "input__ui_only_change=", "input__repo=stackrox/stackrox", "input__pr_number=1", "GITHUB_OUTPUT="+out)
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, string(output))
			data, err := os.ReadFile(out)
			require.NoError(t, err)
			line, _, _ := strings.Cut(string(data), "\n")
			require.True(t, strings.HasPrefix(line, "jobs="))
			var selected map[string]bool
			require.NoError(t, json.Unmarshal([]byte(strings.TrimPrefix(line, "jobs=")), &selected))
			require.Len(t, selected, 9)
			any := false
			for _, run := range selected {
				any = any || run
			}
			require.Equal(t, test.wantAny, any)
		})
	}
}

func TestProwHook(t *testing.T) {
	root := repositoryRoot(t)
	for name, test := range map[string]struct {
		spec, result string
		exit         int
		skip         bool
	}{
		"should skip a verified result":          {spec: `{"type":"presubmit","refs":{"base_sha":"aaa","pulls":[{"sha":"bbb"}]}}`, result: `{"version":1,"job":"gke-qa-e2e-tests","decision":"skip","reason":"docs-only","facts":{"version":1,"complete":true,"every":["docs"]}}`, skip: true},
		"should retain work on resolver failure": {spec: `{"type":"presubmit","refs":{"base_sha":"aaa","pulls":[{"sha":"bbb"}]}}`, exit: 1},
		"should reject a mismatched target":      {spec: `{"type":"presubmit","refs":{"base_sha":"aaa","pulls":[{"sha":"bbb"}]}}`, result: `{"version":1,"job":"other","decision":"skip"}`},
		"should not gate postsubmit":             {spec: `{"type":"postsubmit","refs":{"pulls":[]}}`},
		"should not gate batches":                {spec: `{"type":"presubmit","refs":{"pulls":[{},{}]}}`},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			fake := "#!/usr/bin/env bash\nprintf '%s\\n' \"$DECISION\"\nexit \"$DECISION_EXIT\"\n"
			require.NoError(t, os.WriteFile(filepath.Join(dir, "ci-selection"), []byte(fake), 0o700))
			env := []string{"PATH=" + dir + ":" + os.Getenv("PATH"), "JOB_SPEC=" + test.spec, "DECISION=" + test.result, fmt.Sprintf("DECISION_EXIT=%d", test.exit), "ARTIFACT_DIR=" + dir}
			text, err := runShell(t, "source ci/selection/prow.sh; ci_selection_should_skip gke-qa-e2e-tests", root, env)
			if test.skip {
				require.NoError(t, err, text)
			} else {
				require.Error(t, err, text)
			}
		})
	}
}

func TestSkipOutcome(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repositoryRoot(t), ".openshift-ci/end.sh"))
	require.NoError(t, err)
	start := strings.Index(string(data), "determine_an_overall_job_outcome() {")
	end := strings.Index(string(data)[start:], "\n_JUNIT_CLUSTER_CLASS=")
	function := string(data)[start : start+end]
	for name, test := range map[string]struct{ create, dispatch, destroy, skip, want string }{
		"should pass a skip without cluster setup":       {dispatch: "passed", skip: "true", want: "passed"},
		"should preserve cleanup failure":                {create: "passed", dispatch: "passed", destroy: "failed", skip: "true", want: "failed"},
		"should preserve setup failure":                  {create: "failed", dispatch: "passed", skip: "true", want: "failed"},
		"should preserve cleanup cancellation":           {create: "passed", dispatch: "passed", destroy: "canceled", skip: "true", want: "canceled"},
		"should require cleanup for an existing cluster": {create: "passed", dispatch: "passed", skip: "true", want: "canceled"},
		"should not mask cancellation with a marker":     {dispatch: "canceled", skip: "true", want: "canceled"},
	} {
		t.Run(name, func(t *testing.T) {
			script := "info() { :; }; OUTCOME_PASSED=passed; OUTCOME_FAILED=failed; OUTCOME_CANCELED=canceled;\n" + function + "\ndetermine_an_overall_job_outcome; printf '%s' \"$OVERALL_JOB_OUTCOME\""
			text, err := runShell(t, script, repositoryRoot(t), []string{"CREATE_CLUSTER_OUTCOME=" + test.create, "JOB_DISPATCH_OUTCOME=" + test.dispatch, "DESTROY_CLUSTER_OUTCOME=" + test.destroy, "CI_SELECTION_SKIPPED=" + test.skip})
			require.NoError(t, err, text)
			require.Equal(t, test.want, text)
		})
	}
}

func TestProductionConfiguration(t *testing.T) {
	root := repositoryRoot(t)
	var groups map[string]pathGroup
	require.NoError(t, readJSON(filepath.Join(root, "ci/selection/file-groups.json"), &groups))
	var ignores []string
	require.NoError(t, readJSON(filepath.Join(root, "ci/selection/demo-ignore.json"), &ignores))
	var targets []string
	require.NoError(t, readJSON(filepath.Join(root, "ci/selection/prow-product-tests.json"), &targets))
	facts, err := classify(sampleDiff([]string{"README.md", ".github/workflows/unit-tests.yaml", "ci/selection/main.go"}), groups, ignores)
	require.NoError(t, err)
	if len(ignores) > 0 {
		require.Equal(t, "skip", decide("gke-qa-e2e-tests", facts, targets).Decision)
	} else {
		require.Equal(t, "run", decide("gke-qa-e2e-tests", facts, targets).Decision)
	}
	for _, target := range targets {
		shell := filepath.Join(root, "scripts/ci/jobs", target+".sh")
		python := filepath.Join(root, "scripts/ci/jobs", strings.ReplaceAll(target, "-", "_")+".py")
		_, shellErr := os.Stat(shell)
		_, pythonErr := os.Stat(python)
		require.True(t, shellErr == nil || pythonErr == nil, "enrolled target must have an execution script: %s", target)
	}
	productFacts, err := classify(sampleDiff([]string{"README.md", "central/main.go", "ci/selection/main.go"}), groups, ignores)
	require.NoError(t, err)
	require.Equal(t, "run", decide("gke-qa-e2e-tests", productFacts, targets).Decision)
	require.Equal(t, "run", decide("test-binary-build-commands", facts, targets).Decision)
	facts, err = classify(sampleDiff([]string{"README.md", ".github/workflows/unit-tests.yaml"}), groups, nil)
	require.NoError(t, err)
	require.Empty(t, facts.Every, "production must evaluate the full diff")
	var out bytes.Buffer
	payload, err := json.Marshal(sampleDiff([]string{"README.md"}))
	require.NoError(t, err)
	t.Chdir(root)
	require.NoError(t, run([]string{"--job", "gke-qa-e2e-tests"}, bytes.NewReader(payload), &out))
	require.Contains(t, out.String(), `"decision":"skip"`)
}

func TestDispatchSkipDoesNotProcessMissingTestResults(t *testing.T) {
	root := repositoryRoot(t)
	data, err := os.ReadFile(filepath.Join(root, ".openshift-ci/dispatch.sh"))
	require.NoError(t, err)
	start := strings.Index(string(data), "# Classification gates test work")
	end := strings.Index(string(data)[start:], "\ncase ")
	require.Positive(t, start)
	require.Positive(t, end)
	block := string(data)[start : start+end]
	dir := t.TempDir()
	fake := `#!/usr/bin/env bash
printf '%s\n' '{"version":1,"job":"gke-qa-e2e-tests","decision":"skip","reason":"docs-only","facts":{"version":1,"complete":true,"every":["docs"]}}'
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ci-selection"), []byte(fake), 0o700))
	script := `set -euo pipefail
info() { :; }
set_ci_shared_export() { printf '%s=%s\n' "$1" "$2"; }
ci_job=gke-qa-e2e-tests
OUTCOME_PASSED=passed
trap 'echo UNEXPECTED_TEST_RESULT_PROCESSING' EXIT
` + block
	output, err := runShell(t, script, root, []string{"ROOT=" + root, "PATH=" + dir + ":" + os.Getenv("PATH"), "JOB_SPEC=" + `{"type":"presubmit","refs":{"base_sha":"aaa","pulls":[{"sha":"bbb"}]}}`})
	require.NoError(t, err, output)
	require.Contains(t, output, "CI_SELECTION_SKIPPED=true")
	require.Contains(t, output, "JOB_DISPATCH_OUTCOME=passed")
	require.NotContains(t, output, "UNEXPECTED_TEST_RESULT_PROCESSING")
}

func TestE2EConsumerWiring(t *testing.T) {
	jobs := object(workflow(t, "e2e-dispatch.yaml")["jobs"])
	consumers := 0
	for name, value := range jobs {
		job := object(value)
		uses, _ := job["uses"].(string)
		if !strings.HasPrefix(uses, "./.github/workflows/e2e-") {
			continue
		}
		consumers++
		require.Contains(t, object(job["with"])["should-run"], "fromJSON(needs.should-dispatch.outputs.jobs)."+name)
		require.NotContains(t, fmt.Sprint(job["needs"]), "select-")
	}
	require.Equal(t, 9, consumers)
}

func TestVerificationCannotBeSkippedByProductSelection(t *testing.T) {
	w := workflow(t, "ci-selection.yaml")
	require.Contains(t, object(w["on"]), "pull_request")
	job := object(object(w["jobs"])["verify"])
	require.NotContains(t, job, "needs")
	require.NotContains(t, job, "if")
	guardFound := false
	for _, value := range job["steps"].([]any) {
		step := object(value)
		if step["name"] == "Require full-diff policy outside a draft demo" {
			guardFound = true
			require.Equal(t, "github.event_name != 'pull_request' || !github.event.pull_request.draft", step["if"])
			require.Equal(t, "jq -e 'length == 0' ci/selection/demo-ignore.json", step["run"])
			continue
		}
		require.NotContains(t, step, "if")
	}
	require.True(t, guardFound, "demo exclusions must be rejected outside a draft")
}

func TestActualClassifierHandoff(t *testing.T) {
	root := repositoryRoot(t)
	w := workflow(t, "detect-changes.yml")
	steps := object(object(w["jobs"])["detect"])["steps"].([]any)
	var script string
	for _, value := range steps {
		step := object(value)
		if step["id"] == "classify" {
			script = step["run"].(string)
		}
	}
	require.NotEmpty(t, script)
	dir := t.TempDir()
	payload, err := json.Marshal(sampleDiff([]string{"README.md", "ci/selection/main.go"}))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ci-selection-files.json"), payload, 0o600))
	out := filepath.Join(dir, "output")
	text, err := runShell(t, "set -euo pipefail\n"+script, root, []string{"RUNNER_TEMP=" + dir, "GITHUB_OUTPUT=" + out, "GITHUB_STEP_SUMMARY=" + filepath.Join(dir, "summary")})
	require.NoError(t, err, text)
	encoded, err := os.ReadFile(out)
	require.NoError(t, err)
	var facts fileFacts
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(string(encoded), "facts="))), &facts))
	require.True(t, facts.Complete)
	if len(facts.Ignored) > 0 {
		require.Contains(t, facts.Every, "docs")
	}
}

func TestSkipFinalizerExitStatus(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repositoryRoot(t), ".openshift-ci/end.sh"))
	require.NoError(t, err)
	start := strings.Index(string(data), "end() {")
	end := strings.Index(string(data)[start:], "\ndetermine_an_overall_job_outcome() {")
	function := string(data)[start : start+end]
	for name, outcome := range map[string]string{
		"should accept a successful intentional skip": "passed",
		"should fail when cleanup failed":             "failed",
		"should not accept an interrupted lifecycle":  "canceled",
	} {
		t.Run(name, func(t *testing.T) {
			script := `set -euo pipefail
info() { :; }
openshift_ci_mods() { :; }
openshift_ci_import_creds() { :; }
determine_an_overall_job_outcome() { :; }
generate_cluster_junit() { :; }
save_job_record() { :; }
post_process_test_results() { :; }
send_slack_failure_summary() { :; }
OUTCOME_PASSED=passed
OUTCOME_FAILED=failed
` + function + "\nend\n"
			output, err := runShell(t, script, repositoryRoot(t), []string{"SHARED_DIR=" + t.TempDir(), "CI_SELECTION_SKIPPED=true", "OVERALL_JOB_OUTCOME=" + outcome, "END_SLACK_FAILURE_ATTACHMENTS=unused", "END_JUNIT2JIRA_SUMMARY_FILE=unused"})
			if outcome == "passed" {
				require.NoError(t, err, output)
			} else {
				require.Error(t, err, output)
			}
		})
	}
}
