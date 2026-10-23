package testing_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/nektos/act/pkg/exprparser"
	"github.com/nektos/act/pkg/model"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const unknownFacts = `{"version":1,"complete":false,"any":[],"every":[]}`

type (
	object  = map[string]any
	fixture struct {
		root, binary     string
		workflows        map[string]object
		report, detector object
		local            map[string]selection
		consumers        []string
	}
	selection struct {
		workflow  string
		job, step object
	}
	decision struct{ candidate, enforced, run bool }
	options  struct {
		Event                                   string `json:"event"`
		Draft                                   bool   `json:"draft"`
		Fork                                    bool   `json:"fork"`
		Unknown                                 bool   `json:"unknown"`
		liveLabels                              []string
		labelFailure, failure, stale, staleBase bool
		pages                                   string
		workflow                                string
	}
	collected  struct{ files, uiOnly, facts, runAll, dir string }
	actContext struct {
		Github *model.GithubContext         `json:"github"`
		Env    map[string]string            `json:"env"`
		Steps  map[string]*model.StepResult `json:"steps"`
		Needs  map[string]exprparser.Needs  `json:"needs"`
		Inputs map[string]any               `json:"inputs"`
		Matrix map[string]any               `json:"matrix"`
	}
)

func newFixture(t *testing.T) *fixture {
	t.Helper()
	root, err := filepath.Abs("../../..")
	require.NoError(t, err)
	f := &fixture{root: root, binary: filepath.Join(t.TempDir(), "file-facts"), workflows: map[string]object{}, local: map[string]selection{}}
	command(t, root, "", nil, "go", "build", "-o", f.binary, "./ci/native-selectors/file-facts")
	for _, workflow := range []string{"unit-tests.yaml", "style.yaml", "check-crd-compatibility.yaml", "scanner-db-integration-tests.yaml", "build.yaml", "native-ci-tests.yaml", "e2e-dispatch.yaml"} {
		f.workflows[workflow] = f.workflow(t, workflow)
		for name, value := range obj(f.workflows[workflow]["jobs"]) {
			job := obj(value)
			for _, step := range steps(job) {
				if step["id"] == "selection" {
					if name == "explain-image-build-selection" {
						name = "build-images"
					}
					f.local[name] = selection{workflow: workflow, job: job, step: step}
				}
			}
		}
	}
	f.detector = steps(obj(lookup(f.workflow(t, "detect-changes.yml"), "jobs", "detect")))[0]
	f.report = f.yaml(t, ".github/actions/ci-selection-summary/action.yml")
	for name, value := range f.jobs("e2e-dispatch.yaml") {
		if strings.Contains(str(obj(value)["uses"]), "/e2e-") {
			f.consumers = append(f.consumers, name)
		}
	}
	slices.Sort(f.consumers)
	return f
}

func obj(value any) object {
	if value == nil {
		return object{}
	}
	return value.(map[string]any)
}

func lookup(value any, keys ...string) any {
	for _, key := range keys {
		value = obj(value)[key]
	}
	return value
}

func array(value any) []any {
	if value == nil {
		return nil
	}
	return value.([]any)
}

func str(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func steps(job object) []object {
	result := make([]object, 0, len(array(job["steps"])))
	for _, value := range array(job["steps"]) {
		result = append(result, obj(value))
	}
	return result
}

func encode(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	require.NoError(t, err)
	return string(data)
}

func clone(t *testing.T, value object) object {
	t.Helper()
	var result object
	require.NoError(t, json.Unmarshal([]byte(encode(t, value)), &result))
	return result
}

func digest(value []byte) string { return fmt.Sprintf("%x", sha256.Sum256(value)) }

func (f *fixture) read(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(f.root, name))
	require.NoError(t, err)
	return data
}

func (f *fixture) yaml(t *testing.T, name string) object {
	t.Helper()
	var result object
	require.NoError(t, yaml.Unmarshal(f.read(t, name), &result))
	return result
}

func (f *fixture) json(t *testing.T, name string, result any) {
	t.Helper()
	require.NoError(t, json.Unmarshal(f.read(t, "ci/native-selectors/"+name), result))
}

func (f *fixture) workflow(t *testing.T, name string) object {
	t.Helper()
	return f.yaml(t, ".github/workflows/"+name)
}

func (f *fixture) jobs(name string) object { return obj(f.workflows[name]["jobs"]) }

func command(t *testing.T, dir, input string, env map[string]string, name string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), name, args...)
	cmd.Dir, cmd.Stdin, cmd.Env = dir, strings.NewReader(input), os.Environ()
	for key, value := range env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s %v\n%s", name, args, output)
	return string(output)
}

func evaluate(t *testing.T, source any, context object) string {
	t.Helper()
	text, ok := source.(string)
	if !ok {
		return str(source)
	}
	text = strings.TrimSpace(text)
	if after, ok := strings.CutPrefix(text, "${{"); ok {
		text = strings.TrimSpace(strings.TrimSuffix(after, "}}"))
	}
	var ctx actContext
	require.NoError(t, json.Unmarshal([]byte(encode(t, context)), &ctx))
	interpreter := exprparser.NewInterpeter(&exprparser.EvaluationEnvironment{
		Github: ctx.Github, Env: ctx.Env, Steps: ctx.Steps, Needs: ctx.Needs, Inputs: ctx.Inputs, Matrix: ctx.Matrix,
		Job: &model.JobContext{Status: "success"},
	}, exprparser.Config{Context: "step"})
	value, err := interpreter.Evaluate(text, exprparser.DefaultStatusCheckNone)
	require.NoError(t, err, "expression: %s", text)
	return str(value)
}

func (f *fixture) classify(t *testing.T, paths []string) string {
	t.Helper()
	if paths == nil {
		paths = []string{}
	}
	return strings.TrimSpace(command(t, f.root, encode(t, object{"base": "fixture-base", "head": "fixture-head", "files": paths}), nil, f.binary))
}

func (f *fixture) collect(t *testing.T, paths []string, opt options) collected {
	t.Helper()
	workflow := opt.workflow
	if workflow == "" {
		workflow = "unit-tests.yaml"
	}
	dir := t.TempDir()
	output := filepath.Join(dir, "detected")
	require.NoError(t, os.WriteFile(output, nil, 0o600))
	pages := opt.pages
	if pages == "" {
		entries := make([]object, 0, len(paths))
		for _, name := range paths {
			entries = append(entries, object{"filename": name})
		}
		pages = encode(t, [][]object{entries})
	}
	pageFile := filepath.Join(dir, "pages.json")
	require.NoError(t, os.WriteFile(pageFile, []byte(pages), 0o600))
	mock := `gh() {
  [[ "$FIXTURE_FAILURE" != true ]] || return 1
  if [[ "$2" == */files ]]; then cat "$FIXTURE_PAGES_FILE";
  else printf '%s\t%s' "$FIXTURE_HEAD" "$FIXTURE_BASE"; fi
}
`
	pr, head, base := "1", "fixture-head", "fixture-base"
	if opt.Event != "" && opt.Event != "pull_request" {
		pr = ""
	}
	if opt.stale {
		head = "new-head"
	}
	if opt.staleBase {
		base = "new-base"
	}
	command(t, f.root, "", map[string]string{
		"GH_REPO": "fixture/repo", "GH_TOKEN": "", "PR_NUMBER": pr, "BASE_SHA": "fixture-base", "EXPECTED_HEAD": "fixture-head",
		"RUNNER_TEMP": dir, "FIXTURE_HEAD": head, "FIXTURE_BASE": base, "FIXTURE_FAILURE": strconv.FormatBool(opt.failure), "FIXTURE_PAGES_FILE": pageFile,
		"RUN_ALL":            str(lookup(f.jobs(workflow), "detect-changes", "with", "run_all")),
		"WORKFLOW_FILE_PATH": str(lookup(f.detector, "env", "WORKFLOW_FILE_PATH")), "GITHUB_OUTPUT": output,
	}, bash(), "-c", mock+str(f.detector["run"]))
	data, err := os.ReadFile(output)
	require.NoError(t, err)
	result := collected{facts: unknownFacts, dir: dir}
	text := string(data)
	if _, after, ok := strings.Cut(text, "files<<EOF\n"); ok {
		result.files, _, _ = strings.Cut(after, "\nEOF")
	}
	if strings.Contains(text, "ui_only=true\n") {
		result.uiOnly = "true"
	}
	if strings.Contains(text, "run_all=true\n") {
		result.runAll = "true"
	}
	raw, err := os.ReadFile(filepath.Join(dir, "native-ci-files.json"))
	if err == nil && !opt.Unknown {
		result.facts = strings.TrimSpace(command(t, f.root, string(raw), nil, f.binary))
	} else if !os.IsNotExist(err) {
		require.NoError(t, err)
	}
	return result
}

func bash() string {
	if name := os.Getenv("NATIVE_TEST_BASH"); name != "" {
		return name
	}
	return "bash"
}

func (f *fixture) context(t *testing.T, paths, labels []string, opt options) object {
	t.Helper()
	if labels == nil {
		labels = []string{}
	}
	current := labels
	if opt.liveLabels != nil {
		current = opt.liveLabels
	}
	items := make([]object, 0, len(labels))
	for _, label := range labels {
		items = append(items, object{"name": label})
	}
	event := opt.Event
	if event == "" {
		event = "pull_request"
	}
	files := f.collect(t, paths, opt)
	labelStatus, labelOutput := "success", encode(t, current)
	if opt.labelFailure {
		labelStatus, labelOutput = "failure", ""
	}
	ctx := object{
		"github": object{"event_name": event, "ref": "refs/pull/1/merge", "event": object{"pull_request": object{"labels": items, "draft": opt.Draft, "head": object{"repo": object{"fork": opt.Fork}}}}},
		"needs":  object{"detect-changes": object{"result": "success", "outputs": object{"files": files.files, "ui_only": files.uiOnly, "facts": files.facts, "run_all": files.runAll}}},
		"steps":  object{"labels": object{"outcome": labelStatus, "outputs": object{"labels": labelOutput}}}, "matrix": object{"pg": "16"},
	}
	f.dispatchOutputs(t, ctx)
	return ctx
}

func (f *fixture) dispatchOutputs(t *testing.T, ctx object) {
	t.Helper()
	outputs := object{}
	for key, value := range obj(lookup(f.jobs("e2e-dispatch.yaml"), "should-dispatch", "outputs")) {
		outputs[key] = evaluate(t, value, ctx)
	}
	obj(ctx["needs"])["should-dispatch"] = object{"result": "success", "outputs": outputs}
}

func (f *fixture) explain(t *testing.T, name string, ctx object) decision {
	t.Helper()
	selected := f.local[name]
	context := clone(t, ctx)
	if value := lookup(selected.job, "env", "CI_FILE_FACTS"); value != nil {
		context["env"] = object{"CI_FILE_FACTS": evaluate(t, value, ctx)}
	}
	inputs := object{}
	for key, value := range obj(f.report["inputs"]) {
		inputs[key] = str(obj(value)["default"])
	}
	for key, value := range obj(selected.step["with"]) {
		if key == "note" {
			inputs[key] = value
		} else {
			inputs[key] = evaluate(t, value, context)
		}
	}
	run := evaluate(t, lookup(steps(obj(f.report["runs"]))[0], "env", "RUN"), object{"inputs": inputs})
	return decision{candidate: inputs["candidate"] == "true", enforced: inputs["enforced"] == "true", run: run == "true"}
}

func (f *fixture) candidate(t *testing.T, name string, ctx object) bool {
	t.Helper()
	if _, ok := f.local[name]; ok {
		return f.explain(t, name, ctx).candidate
	}
	ctx = clone(t, ctx)
	obj(ctx["github"])["event_name"] = "pull_request"
	// Candidate checks remove execution restrictions; dispatch tests exercise them separately.
	obj(lookup(ctx, "github", "event", "pull_request", "head", "repo"))["fork"] = false
	outputs := obj(lookup(ctx, "needs", "should-dispatch", "outputs"))
	outputs["enforced"] = "true"
	var labels []string
	require.NoError(t, json.Unmarshal([]byte(str(outputs["labels"])), &labels))
	outputs["labels"] = encode(t, append(labels, "konflux-build"))
	return evaluate(t, lookup(f.jobs("e2e-dispatch.yaml"), name, "with", "should-run"), ctx) == "true"
}

func (f *fixture) dispatch(t *testing.T, ctx object) (map[string]bool, bool, bool) {
	t.Helper()
	jobs := map[string]bool{}
	for _, name := range f.consumers {
		jobs[name] = evaluate(t, lookup(f.jobs("e2e-dispatch.yaml"), name, "with", "should-run"), ctx) == "true"
	}
	fork := lookup(ctx, "github", "event", "pull_request", "head", "repo", "fork") == true
	return jobs, !fork && f.explain(t, "wait-for-images", ctx).run, !fork && f.explain(t, "wait-for-konflux-images", ctx).run
}
