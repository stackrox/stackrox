package testing_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestE2EDecisionsBeforeGates(t *testing.T) {
	f := newFixture(t)
	require.Len(t, f.preparations, 9)
	for _, target := range f.consumers {
		preparation := f.preparations[target]
		require.Equal(t, target, lookup(preparation.step, "with", "target"))
		require.Contains(t, obj(preparation.job["outputs"]), "run-tests")
		require.Contains(t, obj(preparation.job["outputs"]), "report-skipped-checks")
		require.Nil(t, preparation.job["continue-on-error"], "failed preparation must block execution")
		require.Equal(t, "${{ needs.select-"+target+".outputs.run-tests == 'true' }}", lookup(f.jobs("e2e-dispatch.yaml"), target, "with", "should-run"))
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
		"should decide before a failing image waiter":           {files: []string{"go.mod"}, failed: "wait-for-images", kind: "permitted", run: true},
		"should decide before a skipped image waiter":           {files: []string{"go.mod"}, failed: "skipped-image", kind: "permitted", run: true},
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
			obj(ctx["needs"])[prepName] = object{"result": "success", "outputs": f.preparationOutputs(t, test.target, str(result.run), ctx)}
			consumer := obj(f.jobs("e2e-dispatch.yaml")[test.target])
			require.Contains(t, array(consumer["needs"]), prepName)
			require.Equal(t, str(test.run), evaluate(t, lookup(consumer, "with", "should-run"), ctx), "call must consume the emitted execution output")
			if test.failed == "wait-for-images" || test.failed == "skipped-image" {
				require.Equal(t, "false", evaluate(t, consumer["if"], ctx), "an early permission record must not bypass failed images")
			}
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
			require.Equal(t, "run="+str(test.run)+"\ndecision="+test.kind+"\n", string(output))
		})
	}
}

func TestE2ECallerCheckPublication(t *testing.T) {
	f := newFixture(t)
	base := f.context(t, []string{"ci/native-selectors/file-groups.json"}, []string{"ci-dispatcher-enforce"}, options{workflow: "e2e-dispatch.yaml"})
	for _, target := range f.consumers {
		t.Run(target, func(t *testing.T) {
			caller := obj(f.jobs("e2e-dispatch.yaml")[target])
			var waiter string
			for _, dependency := range array(caller["needs"]) {
				if strings.HasPrefix(str(dependency), "wait-for-") {
					waiter = str(dependency)
				}
			}
			require.NotEmpty(t, waiter)
			for name, test := range map[string]struct {
				run, image, imageRun, preparation, failedMetadata, event string
				cancelled, invoke                                        bool
			}{
				"unselected PR publishes skips without images":             {run: "false", image: "success", imageRun: "false", invoke: true},
				"unselected PR publishes skips after skipped image waiter": {run: "false", image: "skipped", invoke: true},
				"unselected PR publishes skips after failed image waiter":  {run: "false", image: "failure", invoke: true},
				"selected PR requires ready images":                        {run: "true", image: "success", imageRun: "true", invoke: true},
				"selected PR rejects unselected images":                    {run: "true", image: "success", imageRun: "false"},
				"selected PR rejects failed images":                        {run: "true", image: "failure", imageRun: "true"},
				"selected PR rejects skipped images":                       {run: "true", image: "skipped", imageRun: "true"},
				"failed preparation is not a selection skip":               {run: "false", image: "success", imageRun: "true", preparation: "failure"},
				"missing preparation output is not a selection skip":       {image: "success", imageRun: "true"},
				"failed detector cannot publish a successful skip":         {run: "false", image: "success", failedMetadata: "detect-changes"},
				"failed dispatcher cannot publish a successful skip":       {run: "false", image: "success", failedMetadata: "should-dispatch"},
				"failed detector cannot request tests":                     {run: "true", image: "success", imageRun: "true", failedMetadata: "detect-changes"},
				"failed dispatcher cannot request tests":                   {run: "true", image: "success", imageRun: "true", failedMetadata: "should-dispatch"},
				"cancellation stops child publication":                     {run: "false", image: "success", cancelled: true},
				"unselected manual event cannot invoke manual defaults":    {run: "false", image: "success", imageRun: "true", event: "workflow_dispatch"},
				"unselected push retains execution guards":                 {run: "false", image: "success", imageRun: "true", event: "push"},
				"selected push retains execution":                          {run: "true", image: "success", imageRun: "true", event: "push", invoke: true},
				"selected manual event retains execution":                  {run: "true", image: "success", imageRun: "true", event: "workflow_dispatch", invoke: true},
			} {
				t.Run(name, func(t *testing.T) {
					ctx := clone(t, base)
					preparation := test.preparation
					if preparation == "" {
						preparation = "success"
					}
					obj(ctx["needs"])[waiter] = object{"result": test.image, "outputs": object{"run": test.imageRun}}
					if test.failedMetadata != "" {
						obj(lookup(ctx, "needs", test.failedMetadata))["result"] = "failure"
					}
					if test.event != "" {
						obj(ctx["github"])["event_name"] = test.event
					}
					if test.cancelled {
						ctx["job"] = object{"status": "cancelled"}
					}
					outputs := f.preparationOutputs(t, target, test.run, ctx)
					obj(ctx["needs"])["select-"+target] = object{"result": preparation, "outputs": outputs}
					require.Equal(t, str(test.invoke), evaluate(t, caller["if"], ctx))
					require.Equal(t, str(test.run == "true" && test.failedMetadata == ""), evaluate(t, lookup(caller, "with", "should-run"), ctx))
					require.False(t, outputs["run-tests"] == "true" && outputs["report-skipped-checks"] == "true", "a preparation cannot request tests and skipped checks together")
				})
			}
		})
	}
}

func TestRequiredE2EChildrenCanReportSkipped(t *testing.T) {
	f := newFixture(t)
	var requirements map[string]map[string][]string
	f.json(t, "testing/required-gha-checks.json", &requirements)
	base := f.context(t, []string{"ci/native-selectors/file-groups.json"}, []string{"ci-dispatcher-enforce"}, options{workflow: "e2e-dispatch.yaml"})
	checked := 0
	for caller, checks := range requirements["e2e-dispatch.yaml"] {
		for _, check := range checks {
			parts := strings.Split(check, " / ")
			if len(parts) == 1 {
				continue
			}
			t.Run(check, func(t *testing.T) {
				ctx := clone(t, base)
				job := obj(f.jobs("e2e-dispatch.yaml")[caller])
				for _, dependency := range array(job["needs"]) {
					obj(ctx["needs"])[str(dependency)] = object{"result": "success", "outputs": object{"run": "false"}}
				}
				obj(ctx["needs"])["select-"+caller] = object{"result": "success", "outputs": f.preparationOutputs(t, caller, "false", ctx)}
				require.Equal(t, "true", evaluate(t, job["if"], ctx), "the caller must instantiate its child checks")
				for i, part := range parts[1:] {
					child := f.workflow(t, filepath.Base(str(job["uses"])))
					require.Equal(t, "boolean", lookup(child, "on", "workflow_call", "inputs", "should-run", "type"))
					require.Equal(t, "false", evaluate(t, lookup(job, "with", "should-run"), ctx), "each reusable-workflow boundary must preserve the rejection")
					ctx["inputs"], ctx["needs"] = object{"should-run": false}, object{}
					job = obj(lookup(child, "jobs", part))
					require.NotEmpty(t, job, "required check component %s", part)
					for _, dependency := range arrayNeeds(job["needs"]) {
						prerequisite := obj(lookup(child, "jobs", str(dependency)))
						require.Equal(t, "false", evaluate(t, prerequisite["if"], ctx), "a skipped suite must not provision its prerequisites")
						obj(ctx["needs"])[str(dependency)] = object{"result": "skipped", "outputs": object{}}
					}
					condition := job["if"]
					if condition == nil {
						condition = "success()"
					}
					require.Equal(t, str(i < len(parts)-2), evaluate(t, condition, ctx), "intermediate calls must expand; required leaf jobs must skip")
				}
			})
			checked++
		}
	}
	require.Equal(t, 4, checked)
}
