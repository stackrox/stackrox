package testing_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func facts(t *testing.T, text string) object {
	t.Helper()
	var value object
	require.NoError(t, json.Unmarshal([]byte(text), &value))
	return value
}

func TestCollector(t *testing.T) {
	f := newFixture(t)
	t.Run("UI-only and mixed files", func(t *testing.T) {
		ui := f.collect(t, []string{"ui/App.tsx"}, options{})
		require.Equal(t, "^ui/App.tsx$", ui.files)
		require.Equal(t, "true", ui.uiOnly)
		require.Contains(t, facts(t, ui.facts)["every"], "ui")
		require.Empty(t, f.collect(t, []string{"ui/App.tsx", "README.md"}, options{}).uiOnly)
	})
	t.Run("pagination and deleted paths", func(t *testing.T) {
		result := f.collect(t, nil, options{pages: `[[{"filename":"ui/App.tsx"}],[{"filename":"sensor/old.go","status":"removed"}]]`})
		require.Contains(t, result.files, "^sensor/old.go$")
	})
	t.Run("renames retain previous paths", func(t *testing.T) {
		result := f.collect(t, nil, options{pages: `[[{"filename":"ui/new.tsx","previous_filename":"sensor/kubernetes/client/old.go"}]]`})
		require.Contains(t, result.files, "^sensor/kubernetes/client/old.go$")
		require.NotContains(t, facts(t, result.facts)["every"], "ui")
	})
	t.Run("large mixed diff and legacy overrides", func(t *testing.T) {
		many := make([]string, 1000)
		for i := range many {
			many[i] = "ui/" + strconv.Itoa(i) + "/" + strings.Repeat("x", 90) + ".tsx"
		}
		require.Empty(t, f.collect(t, append([]string{"central/main.go"}, many...), options{}).uiOnly)
		override := f.collect(t, append([]string{"Makefile"}, many...), options{})
		require.Empty(t, override.files)
		require.Equal(t, "true", override.runAll)
		require.Equal(t, true, facts(t, override.facts)["complete"], "run_all must retain classified facts")
	})
	t.Run("known empty diff is complete", func(t *testing.T) {
		require.Equal(t, true, facts(t, f.collect(t, nil, options{}).facts)["complete"])
	})
	t.Run("detector edits do not hide explicit product build overrides", func(t *testing.T) {
		result := f.collect(t, []string{".github/workflows/detect-changes.yml", "Makefile"}, options{})
		require.Equal(t, "true", result.runAll)
		require.Equal(t, true, facts(t, result.facts)["complete"])
	})
	paths := make([]string, 3000)
	for i := range paths {
		paths[i] = "ui/" + strconv.Itoa(i) + ".tsx"
	}
	tests := map[string]struct {
		paths []string
		opt   options
	}{
		"invalid page structure":             {opt: options{pages: `[{"error":true}]`}},
		"empty filename":                     {paths: []string{""}},
		"API failure":                        {paths: []string{"README.md"}, opt: options{failure: true}},
		"stale head":                         {paths: []string{"README.md"}, opt: options{stale: true}},
		"stale base":                         {paths: []string{"README.md"}, opt: options{staleBase: true}},
		"non-PR event":                       {paths: []string{"README.md"}, opt: options{Event: "push"}},
		"newline in filename":                {paths: []string{"ui/a\nb.tsx"}},
		"carriage return in filename":        {paths: []string{"ui/a\rb.tsx"}},
		"3000 files exceed the API boundary": {paths: paths},
		"oversized filename output":          {paths: []string{"ui/" + strings.Repeat("a", 200001)}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result := f.collect(t, test.paths, test.opt)
			require.Empty(t, result.files)
			require.Empty(t, result.uiOnly)
			require.Empty(t, result.runAll)
			require.Equal(t, unknownFacts, result.facts)
		})
	}
}

func TestClassifierHandoff(t *testing.T) {
	f := newFixture(t)
	workflow := f.workflow(t, "detect-changes.yml")
	job := obj(lookup(workflow, "jobs", "detect"))
	var classify object
	for _, step := range steps(job) {
		if step["id"] == "classify" {
			classify = step
		}
	}
	require.NotEmpty(t, classify)
	collected := f.collect(t, []string{"ui/App.tsx", "sensor/main.go"}, options{})
	output := filepath.Join(collected.dir, "classified")
	command(t, f.root, "", map[string]string{
		"RUNNER_TEMP": collected.dir, "GITHUB_OUTPUT": output,
		"GITHUB_STEP_SUMMARY": filepath.Join(collected.dir, "summary"),
	}, bash(), "-c", str(classify["run"]))
	data, err := os.ReadFile(output)
	require.NoError(t, err)
	reported, ok := strings.CutPrefix(strings.TrimSpace(string(data)), "facts=")
	require.True(t, ok)
	require.JSONEq(t, collected.facts, reported)
	for name, test := range map[string]struct {
		outputs object
		want    string
	}{
		"publishes classifier output": {outputs: object{"facts": reported}, want: reported},
		"missing output falls back":   {outputs: object{}, want: unknownFacts},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := object{"steps": object{str(classify["id"]): object{"outputs": test.outputs}}}
			require.JSONEq(t, test.want, evaluate(t, lookup(job, "outputs", "facts"), ctx))
		})
	}
	require.Equal(t, "${{ jobs.detect.outputs.facts }}", lookup(workflow, "on", "workflow_call", "outputs", "facts", "value"))
	require.Equal(t, "${{ steps.list.outputs.run_all }}", lookup(job, "outputs", "run_all"))
	require.Equal(t, "${{ jobs.detect.outputs.run_all }}", lookup(workflow, "on", "workflow_call", "outputs", "run_all", "value"))
}

func TestFileClassifier(t *testing.T) {
	f := newFixture(t)
	var groups map[string]struct {
		Include, Exclude []string
	}
	f.json(t, "file-groups.json", &groups)
	require.Len(t, groups, 20)
	paths := map[string][]string{
		"empty": {}, "UI only": {"ui/a.tsx"}, "mixed UI": {"ui/a.tsx", "OWNERS"},
		"Scanner only": {"scanner/config.yaml"}, "mixed Scanner": {"scanner/config.yaml", "README.md"},
		"Central and Sensor": {"central/sac/store.go", "sensor/kubernetes/client/client.go"},
		"shared dependency":  {"pkg/concurrency/context.go"}, "proto outside known directories": {"elsewhere/schema.proto"},
		"Go-like extension": {"elsewhere/foo.go.txt"}, "JSON filename encoding": {"line\nbreak.go"},
		"CI Go and docs":        {"ci/native-selectors/file-facts/main.go", "README.md"},
		"CI and product Go":     {"ci/native-selectors/file-facts/main.go", "central/sac/store.go", "README.md"},
		"roxvet and product Go": {"tools/roxvet/roxvet.go", "sensor/common/centralclient/client.go"},
	}
	for name, paths := range paths {
		t.Run(name, func(t *testing.T) {
			result := facts(t, f.classify(t, paths))
			for group, patterns := range groups {
				matches := func(file string) bool {
					match := func(pattern string) bool { return regexp.MustCompile(pattern).MatchString(file) }
					return slices.ContainsFunc(patterns.Include, match) && !slices.ContainsFunc(patterns.Exclude, match)
				}
				require.Equal(t, slices.ContainsFunc(paths, matches), slices.Contains(array(result["any"]), any(group)), "any/%s", group)
				require.Equal(t, len(paths) > 0 && !slices.ContainsFunc(paths, func(file string) bool { return !matches(file) }), slices.Contains(array(result["every"]), any(group)), "every/%s", group)
			}
		})
	}
	for _, input := range []string{"null", "{}", `{"base":"a","head":"b","files":null}`, `{"base":"a","head":"b","files":[""]}`, `{"base":"a","head":"b","files":[3]}`, `{"base":"a","head":"b","files":[]} {}`} {
		t.Run("reject input "+input, func(t *testing.T) {
			cmd := exec.CommandContext(t.Context(), f.binary)
			cmd.Dir = f.root
			cmd.Stdin = strings.NewReader(input)
			_, err := cmd.CombinedOutput()
			require.Error(t, err)
		})
	}
	for _, groups := range []string{`{"invalid":{"include":["["]}}`, `{"invalid":{"include":[".*"],"exclude":["["]}}`, `null`, `{}`, `{"empty":{"include":[]}}`, `{"":{"include":["^ui/"]}}`} {
		t.Run("reject groups "+groups, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "groups.json")
			require.NoError(t, os.WriteFile(file, []byte(groups), 0o600))
			cmd := exec.CommandContext(t.Context(), f.binary, "--groups", file)
			cmd.Dir = f.root
			cmd.Stdin = strings.NewReader(`{"base":"a","head":"b","files":[]}`)
			_, err := cmd.CombinedOutput()
			require.Error(t, err)
		})
	}
}
