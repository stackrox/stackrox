package testing_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkflowPreservation(t *testing.T) {
	f := newFixture(t)
	var baseline struct {
		Base, Serialization string
		Jobs                map[string]struct {
			Structure, Steps string
			Conditions       []string
			If, Needs        any
		}
		Headers, UnchangedJobs, Children map[string]string
	}
	f.json(t, "testing/base-workflows.json", &baseline)
	require.Equal(t, "74bfddc24b07e120a1522f2df8cf215c4ae0812c", baseline.Base)
	require.Equal(t, "sorted-key JSON with Go encoding/json escaping", baseline.Serialization)
	gates := 0
	for key, saved := range baseline.Jobs {
		t.Run(key, func(t *testing.T) {
			file, name, ok := strings.Cut(key, "/")
			require.True(t, ok)
			job := clone(t, obj(f.jobs(file)[name]))
			var commands []object
			var preserved []object
			for _, step := range steps(job) {
				if step["id"] == "selection" || step["id"] == "selection-record" {
					continue
				}
				preserved = append(preserved, step)
				copy := clone(t, step)
				delete(copy, "if")
				commands = append(commands, copy)
			}
			delete(job, "steps")
			waiter := strings.HasPrefix(name, "wait-for-")
			if waiter {
				require.Equal(t, []any{"detect-changes", "should-dispatch"}, job["needs"])
				require.Equal(t, "false", evaluate(t, job["if"], f.context(t, nil, nil, options{Fork: true})))
				job["needs"], job["if"] = saved.Needs, saved.If
				delete(obj(job["outputs"]), "run")
			}
			if env, ok := job["env"]; ok {
				delete(obj(env), "CI_FILE_FACTS")
				if len(obj(env)) == 0 {
					delete(job, "env")
				}
			}
			if saved.Needs == nil {
				delete(job, "needs")
			}
			require.Equal(t, saved.Structure, digest([]byte(encode(t, job))), "matrix, outer guard, check name or environment")
			require.Equal(t, saved.Steps, digest([]byte(encode(t, commands))), "execution commands")
			ctx := f.context(t, []string{"README.md"}, nil, options{})
			for _, run := range []string{"false", "true", ""} {
				obj(ctx["steps"])["selection"] = object{"outputs": object{"run": run}}
				for i, step := range preserved {
					gated := waiter && i == len(preserved)-1 || !waiter && i > 0 && name != "style-check" && name != "define-job-matrix"
					want := evaluate(t, saved.Conditions[i], ctx)
					if gated && run == "false" {
						want = "false"
					}
					expression := step["if"]
					if expression == nil {
						expression = "success()"
					}
					require.Equal(t, want, evaluate(t, expression, ctx), "step %d, run=%q", i, run)
					gates++
				}
			}
			if file != "build.yaml" {
				types := array(lookup(f.workflows[file], "on", "pull_request", "types"))
				require.Contains(t, types, "labeled")
				require.Contains(t, types, "unlabeled")
			}
		})
	}
	for file, hash := range baseline.Headers {
		t.Run("header "+file, func(t *testing.T) {
			header := clone(t, f.workflows[file])
			delete(header, "jobs")
			pull := obj(lookup(header, "on", "pull_request"))
			filtered := []any{}
			for _, event := range array(pull["types"]) {
				if event != "labeled" && event != "unlabeled" {
					filtered = append(filtered, event)
				}
			}
			pull["types"] = filtered
			require.Equal(t, hash, digest([]byte(encode(t, header))), "non-PR triggers or permissions")
			require.Nil(t, pull["paths"])
			require.Nil(t, pull["paths-ignore"])
		})
	}
	for key, hash := range baseline.UnchangedJobs {
		file, name, _ := strings.Cut(key, "/")
		if key == "unit-tests.yaml/detect-changes" {
			require.Equal(t, object{
				"uses": "./.github/workflows/detect-changes.yml",
				"with": object{"run_all": "^Makefile$\n^make/\n^.github/actions/job-preamble/\n"},
			}, f.jobs(file)[name], "only product-build overrides belong to this caller")
			continue
		}
		require.Equal(t, hash, digest([]byte(encode(t, f.jobs(file)[name]))), key)
	}
	for file, hash := range baseline.Children {
		require.Equal(t, hash, digest(f.read(t, ".github/workflows/"+file)), file)
	}
	require.Equal(t, 438, gates)
	t.Logf("%d preserved-step checks", gates)
}

func TestRequiredChecksAndImageGates(t *testing.T) {
	f := newFixture(t)
	matrix := obj(f.jobs("build.yaml")["define-job-matrix"])
	require.Nil(t, matrix["needs"], "image production must not wait for classification")
	require.Nil(t, lookup(matrix, "env", "CI_FILE_FACTS"))
	report := obj(f.jobs("build.yaml")["explain-image-build-selection"])
	require.Equal(t, "detect-changes", report["needs"])
	require.Nil(t, report["outputs"], "the advisory report must not control image production")
	var requirements map[string]map[string][]string
	f.json(t, "testing/fixtures/required-gha-checks.json", &requirements)
	contexts := 0
	for file, jobs := range requirements {
		for name, checks := range jobs {
			require.Contains(t, f.jobs(file), name)
			contexts += len(checks)
		}
	}
	require.Equal(t, 22, contexts)
	for _, name := range f.consumers {
		job := obj(f.jobs("e2e-dispatch.yaml")[name])
		child := f.workflow(t, filepath.Base(str(job["uses"])))
		require.Equal(t, "boolean", lookup(child, "on", "workflow_call", "inputs", "should-run", "type"))
		needs := object{}
		for _, name := range array(job["needs"]) {
			needs[str(name)] = object{"outputs": object{"run": "false"}}
		}
		require.Equal(t, "false", evaluate(t, job["if"], object{"needs": needs}), "consumer %s", name)
	}
	require.NoFileExists(t, filepath.Join(f.root, ".github/workflows/native-ci-selection.yaml"))
	require.NoFileExists(t, filepath.Join(f.root, "ci/native-selectors/path_facts.py"))
}

func TestBuildMatrices(t *testing.T) {
	f := newFixture(t)
	var script string
	for _, step := range steps(obj(f.jobs("build.yaml")["define-job-matrix"])) {
		if step["id"] == "define-job-matrix" {
			script = str(step["run"])
		}
	}
	require.NotEmpty(t, script)
	for name, test := range map[string]struct{ pr, tag, labels string }{
		"default PR":         {pr: "true", tag: "false"},
		"PR build overrides": {pr: "true", tag: "false", labels: "ci-build-all-arch,ci-build-prerelease,ci-build-race-condition-debug"},
		"non-PR":             {pr: "false", tag: "false"},
		"tagged":             {pr: "false", tag: "true"},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(dir, "scripts/ci"), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "scripts/ci/lib.sh"), []byte(`is_in_PR_context() { [[ "$FIXTURE_PR" == true ]]; }
is_tagged() { [[ "$FIXTURE_TAG" == true ]]; }
pr_has_label() { [[ ",$FIXTURE_LABELS," == *",$1,"* ]]; }
`), 0o600))
			output := filepath.Join(dir, "output")
			require.NoError(t, os.WriteFile(output, nil, 0o600))
			command(t, dir, "", map[string]string{"GITHUB_OUTPUT": output, "FIXTURE_PR": test.pr, "FIXTURE_TAG": test.tag, "FIXTURE_LABELS": test.labels}, bash(), "-c", script)
			data, err := os.ReadFile(output)
			require.NoError(t, err)
			var matrix object
			require.NoError(t, json.Unmarshal([]byte(strings.TrimPrefix(strings.TrimSpace(string(data)), "matrix=")), &matrix))
			main := obj(matrix["build_and_push_main"])
			require.Contains(t, main["arch"], "amd64")
			require.Contains(t, main["arch"], "arm64")
			require.Contains(t, main["name"], "STACKROX_BRANDING")
			require.Contains(t, main["name"], "RHACS_BRANDING")
			if test.pr == "false" || test.labels != "" {
				require.Contains(t, main["arch"], "s390x")
			}
		})
	}
}
