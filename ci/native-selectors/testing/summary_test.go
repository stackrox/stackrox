package testing_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSelectionDecisionLogging(t *testing.T) {
	f := newFixture(t)
	step := steps(obj(f.report["runs"]))[0]
	for name, test := range map[string]struct {
		candidate, eligible, enforced, run bool
		decision                           string
	}{
		"selected under enforcement":               {candidate: true, eligible: true, enforced: true, run: true, decision: "permitted"},
		"unselected under enforcement":             {eligible: true, enforced: true, decision: "skipped-by-selection"},
		"selected but blocked under enforcement":   {candidate: true, enforced: true, decision: "blocked-by-execution"},
		"unselected and blocked under enforcement": {enforced: true, decision: "blocked-by-execution"},
		"selected in shadow":                       {candidate: true, eligible: true, run: true, decision: "permitted"},
		"unselected in shadow":                     {eligible: true, run: true, decision: "permitted"},
		"selected but blocked in shadow":           {candidate: true, decision: "blocked-by-execution"},
		"unselected and blocked in shadow":         {decision: "blocked-by-execution"},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			inputs := object{"inputs": object{"candidate": str(test.candidate), "eligible": str(test.eligible), "enforced": str(test.enforced), "note": ""}}
			env := map[string]string{"GITHUB_OUTPUT": filepath.Join(dir, "output"), "GITHUB_STEP_SUMMARY": filepath.Join(dir, "summary")}
			for key, value := range obj(step["env"]) {
				env[key] = evaluate(t, value, inputs)
			}
			output := command(t, f.root, "", env, bash(), "-c", str(step["run"]))
			require.Contains(t, output, "STACKROX_CI_SELECTION_V1 "+test.decision+"\n")
			data, err := os.ReadFile(env["GITHUB_OUTPUT"])
			require.NoError(t, err)
			require.Equal(t, "run="+str(test.run)+"\ndecision="+test.decision+"\n", string(data), "reporting must preserve the execution output")
			context := object{"inputs": inputs["inputs"], "github": object{"job": "go"}, "steps": object{"report": object{"outputs": object{"run": str(test.run), "decision": test.decision}}}}
			outputs := object{}
			for key, value := range obj(f.report["outputs"]) {
				outputs[key] = evaluate(t, lookup(value, "value"), context)
			}
			context["steps"] = object{"selection": object{"outputs": outputs}}
			var marker object
			for _, value := range steps(obj(f.jobs("unit-tests.yaml")["go"])) {
				if value["id"] == "selection-record" {
					marker = value
				}
			}
			require.NotNil(t, marker)
			pattern := regexp.MustCompile(`\$\{\{.*?\}\}`)
			name := pattern.ReplaceAllStringFunc(str(marker["name"]), func(expression string) string { return evaluate(t, expression, context) })
			require.Equal(t, "CI selection: "+test.decision+" | go | candidate="+str(test.candidate)+" | eligible="+str(test.eligible)+" | enforced="+str(test.enforced), name)
		})
	}
}

func TestLiveSelectionObserverIntegration(t *testing.T) {
	f := newFixture(t)
	require.Len(t, f.jobs("ci-selection-report.yaml"), 1, "one observer owns the shared comment")
	observer := obj(f.jobs("ci-selection-report.yaml")["selection-report"])
	require.ElementsMatch(t, []any{"opened", "reopened", "synchronize", "ready_for_review", "labeled", "unlabeled"}, lookup(f.workflows["ci-selection-report.yaml"], "on", "pull_request", "types"))
	require.Nil(t, observer["needs"], "the observer must start without waiting for decisions or tests")
	require.Equal(t, true, observer["continue-on-error"])
	require.Equal(t, object{"actions": "read", "contents": "read", "pull-requests": "write"}, observer["permissions"])
	require.Equal(t, object{"group": "ci-selection-comment-${{ github.event.pull_request.number }}", "cancel-in-progress": true}, observer["concurrency"])
	require.Equal(t, 12, observer["timeout-minutes"])
	observerSteps := steps(observer)
	observe := observerSteps[len(observerSteps)-1]
	require.Contains(t, str(observe["run"]), "--watch 10m --interval 5s --post")
	require.Equal(t, "${{ github.token }}", lookup(observe, "env", "GITHUB_TOKEN"))
	for name, test := range map[string]struct {
		event, repo string
		pr          int
		fork, want  bool
	}{
		"trusted prototype PR": {event: "pull_request", repo: "stackrox/stackrox", pr: 23602, want: true},
		"fork":                 {event: "pull_request", repo: "stackrox/stackrox", pr: 23602, fork: true},
		"other PR":             {event: "pull_request", repo: "stackrox/stackrox", pr: 1},
		"other repository":     {event: "pull_request", repo: "fork/repo", pr: 23602},
		"non-PR event":         {event: "workflow_dispatch", repo: "stackrox/stackrox", pr: 23602},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := object{"github": object{"event_name": test.event, "repository": test.repo, "event": object{"pull_request": object{"number": test.pr, "head": object{"repo": object{"fork": test.fork}}}}}}
			require.Equal(t, str(test.want), evaluate(t, observer["if"], ctx))
		})
	}
	for file := range f.workflows {
		for name, value := range f.jobs(file) {
			job := obj(value)
			require.NotEqual(t, "report-ci-timing", name)
			for _, dependency := range arrayNeeds(job["needs"]) {
				require.NotEqual(t, "selection-report", dependency, "tests must not depend on reporting")
			}
			items := steps(job)
			for i, step := range items {
				if step["id"] == "selection" {
					require.Less(t, i+1, len(items))
					record := items[i+1]
					require.Equal(t, "selection-record", record["id"], "%s/%s", file, name)
					require.Equal(t, ":", record["run"], "recording must not evaluate selection rules")
					require.True(t, strings.HasPrefix(str(record["name"]), "CI selection: ${{ steps.selection.outputs.decision"))
				}
			}
		}
	}
	for _, preparation := range f.preparations {
		require.ElementsMatch(t, []any{"detect-changes", "should-dispatch"}, array(preparation.job["needs"]), "E2E decisions must not wait for images")
	}
	files, err := os.ReadDir(filepath.Join(f.root, ".github/workflows"))
	require.NoError(t, err)
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".yml") || strings.HasSuffix(file.Name(), ".yaml") {
			require.NotContains(t, string(f.read(t, ".github/workflows/"+file.Name())), "report-ci-timing")
		}
	}
}

func arrayNeeds(value any) []any {
	if name, ok := value.(string); ok {
		return []any{name}
	}
	return array(value)
}
