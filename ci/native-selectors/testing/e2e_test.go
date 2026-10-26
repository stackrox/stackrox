package testing_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestE2EDecisionsBeforeGates(t *testing.T) {
	f := newFixture(t)
	require.Len(t, f.preparations, 9)
	for _, target := range f.consumers {
		preparation := f.preparations[target]
		require.Equal(t, target, lookup(preparation.step, "with", "target"))
		require.Equal(t, "${{ steps.selection.outputs.run }}", lookup(preparation.job, "outputs", "run"))
		require.Nil(t, preparation.job["continue-on-error"], "failed preparation must block execution")
		require.Equal(t, "${{ needs.select-"+target+".outputs.run == 'true' }}", lookup(f.jobs("e2e-dispatch.yaml"), target, "with", "should-run"))
	}
	for name, test := range map[string]struct {
		target, kind, failed string
		files, labels        []string
		opt                  options
		run                  bool
	}{
		"should record an unrelated change as a selection skip": {kind: "skipped-by-selection"},
		"should record a mixed broad change as permitted":       {files: []string{"go.mod", "README.md"}, kind: "permitted", run: true},
		"should record an explicit label as permitted":          {labels: []string{"e2e-byodb-tests"}, kind: "permitted", run: true},
		"should retain run-all's image restriction":             {target: "e2e-qa-tests-gke-konflux", labels: []string{"ci-run-all-tests"}, kind: "blocked-by-execution"},
		"should permit run-all with images":                     {target: "e2e-qa-tests-gke-konflux", labels: []string{"ci-run-all-tests", "konflux-build"}, kind: "permitted", run: true},
		"should retain fork restrictions":                       {labels: []string{"ci-run-all-tests"}, opt: options{Fork: true}, kind: "blocked-by-execution"},
		"should permit an enforced draft":                       {labels: []string{"e2e-byodb-tests"}, opt: options{Draft: true}, kind: "permitted", run: true},
		"should retain shadow draft restrictions":               {target: "e2e-gke-upgrade-tests", opt: options{Draft: true, liveLabels: []string{}}, kind: "blocked-by-execution"},
		"should retain shadow ready-PR execution":               {target: "e2e-gke-upgrade-tests", opt: options{liveLabels: []string{}}, kind: "permitted", run: true},
		"should retain the QA unavailable-file default":         {target: "e2e-qa-tests-gke", opt: options{Unknown: true}, kind: "skipped-by-selection"},
		"should retain the BYODB unavailable-file default":      {opt: options{Unknown: true}, kind: "permitted", run: true},
		"should permit a non-PR event":                          {opt: options{Event: "push"}, kind: "permitted", run: true},
		"should record an image-wait failure":                   {files: []string{"go.mod"}, failed: "wait-for-images", kind: "blocked-by-execution"},
		"should record a skipped image prerequisite":            {files: []string{"go.mod"}, failed: "skipped-image", kind: "blocked-by-execution"},
		"should record a detector job failure":                  {failed: "detect-changes", kind: "blocked-by-execution"},
		"should record a label-collection job failure":          {failed: "should-dispatch", kind: "blocked-by-execution"},
	} {
		t.Run(name, func(t *testing.T) {
			if test.target == "" {
				test.target = "e2e-byodb-tests"
			}
			if test.files == nil {
				test.files = []string{"README.md"}
			}
			ctx := f.context(t, test.files, append(test.labels, "ci-dispatcher-enforce"), test.opt)
			for _, waiter := range []string{"wait-for-images", "wait-for-konflux-images"} {
				obj(ctx["needs"])[waiter] = object{"result": "success", "outputs": object{"run": str(f.explain(t, waiter, ctx).run)}}
			}
			if test.failed != "" {
				failed, result := test.failed, "failure"
				if failed == "skipped-image" {
					failed, result = "wait-for-images", "skipped"
				}
				obj(ctx["needs"])[failed] = object{"result": result, "outputs": object{}}
			}
			preparation := f.preparations[test.target]
			require.Equal(t, "true", evaluate(t, preparation.job["if"], ctx), "preparation must survive failed/skipped dependencies")
			result := f.explain(t, test.target, ctx)
			require.Equal(t, test.kind, result.kind)
			require.Equal(t, test.run, result.run)
			prepName := "select-" + test.target
			obj(ctx["needs"])[prepName] = object{"result": "success", "outputs": object{"run": str(result.run)}}
			consumer := obj(f.jobs("e2e-dispatch.yaml")[test.target])
			require.Contains(t, array(consumer["needs"]), prepName)
			require.Equal(t, str(test.run), evaluate(t, lookup(consumer, "with", "should-run"), ctx), "call must consume the emitted execution output")
			obj(ctx["needs"])[prepName] = object{"result": "failure", "outputs": object{}}
			require.Equal(t, "false", evaluate(t, consumer["if"], ctx), "a preparation failure must block the call")

			inputs := object{}
			for key, value := range obj(preparation.step["with"]) {
				if key == "candidate" || key == "eligible" || key == "enforced" {
					inputs[key] = evaluate(t, value, ctx)
				} else {
					inputs[key] = value
				}
			}
			dir := t.TempDir()
			env := map[string]string{"GITHUB_OUTPUT": filepath.Join(dir, "output"), "GITHUB_STEP_SUMMARY": filepath.Join(dir, "summary")}
			script := steps(obj(f.report["runs"]))[0]
			for key, value := range obj(script["env"]) {
				env[key] = evaluate(t, value, object{"inputs": inputs})
			}
			log := command(t, f.root, "", env, bash(), "-c", str(script["run"]))
			require.Contains(t, log, "STACKROX_CI_SELECTION_V1 "+test.kind+"\n")
			require.Contains(t, log, "STACKROX_CI_SELECTION_TARGET_V1 "+test.target+"\n")
			output, err := os.ReadFile(env["GITHUB_OUTPUT"])
			require.NoError(t, err)
			require.Equal(t, "run="+str(test.run)+"\n", string(output))
		})
	}
}
