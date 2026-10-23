package testing_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTimingWorkflowIntegration(t *testing.T) {
	f := newFixture(t)
	files, err := os.ReadDir(filepath.Join(f.root, ".github/workflows"))
	require.NoError(t, err)
	count := 0
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".yaml") && !strings.HasSuffix(file.Name(), ".yml") {
			continue
		}
		workflow := f.yaml(t, ".github/workflows/"+file.Name())
		events, ok := workflow["on"].(map[string]any)
		if !ok {
			continue
		}
		if _, ok := events["pull_request"]; !ok {
			continue
		}
		count++
		t.Run(file.Name(), func(t *testing.T) {
			jobs := obj(workflow["jobs"])
			reporter := obj(jobs["report-ci-timing"])
			require.Equal(t, "./.github/workflows/report-ci-timing.yaml", reporter["uses"])
			require.Equal(t, "CI timing report", reporter["name"])
			var dependencies []any
			for name := range jobs {
				if name != "report-ci-timing" {
					dependencies = append(dependencies, name)
				}
			}
			require.ElementsMatch(t, dependencies, array(reporter["needs"]), "the reporter must wait for every other job")
			require.Equal(t, "${{ always() && github.event_name == 'pull_request' && !github.event.pull_request.head.repo.fork }}", reporter["if"])
			require.Equal(t, object{"actions": "read", "contents": "read", "pull-requests": "write"}, reporter["permissions"])
			require.Nil(t, reporter["secrets"], "publishing uses the automatic bot token, not an inherited personal token")
		})
	}
	require.NotZero(t, count)
	t.Logf("All %d PR workflows wait for their own jobs and call the shared bot reporter", count)

	shared := f.yaml(t, ".github/workflows/report-ci-timing.yaml")
	require.Equal(t, object{"workflow_call": nil}, shared["on"], "reporting must work from a PR branch without default-branch event triggers")
	publish := obj(lookup(shared, "jobs", "publish"))
	require.Equal(t, true, publish["continue-on-error"], "reporting must not fail the test workflow")
	require.Equal(t, "github.event_name == 'pull_request' && github.repository == 'stackrox/stackrox' && github.event.pull_request.number == 23602 && !github.event.pull_request.head.repo.fork && (github.event.action == 'opened' || github.event.action == 'synchronize') && github.run_attempt == '1'", publish["if"])
	require.Equal(t, object{
		"group":              "ci-timing-comment-${{ github.event.pull_request.number }}",
		"cancel-in-progress": false, "queue": "max",
	}, publish["concurrency"], "serialize creation and updates without canceling pending callers")
	for _, step := range steps(publish) {
		if strings.HasPrefix(str(step["uses"]), "actions/checkout@") {
			require.Equal(t, false, lookup(step, "with", "persist-credentials"))
		}
		if step["name"] == "Update timing comment" {
			require.Equal(t, "${{ github.token }}", lookup(step, "env", "GITHUB_TOKEN"))
			require.Equal(t, `go run ./ci/native-selectors/timing-report --repo "$GITHUB_REPOSITORY" --pr "$PR_NUMBER" --run "$GITHUB_RUN_ID" --post`, step["run"])
		}
	}
	for name, test := range map[string]struct {
		action, attempt, event string
		repo                   string
		pr                     int
		fork, want             bool
	}{
		"should report the initial PR opening":            {action: "opened", attempt: "1", event: "pull_request", want: true},
		"should report each new commit":                   {action: "synchronize", attempt: "1", event: "pull_request", want: true},
		"should ignore label additions":                   {action: "labeled", attempt: "1", event: "pull_request"},
		"should ignore label removals":                    {action: "unlabeled", attempt: "1", event: "pull_request"},
		"should ignore manual reruns of an opening event": {action: "opened", attempt: "2", event: "pull_request"},
		"should ignore manual reruns of a commit event":   {action: "synchronize", attempt: "2", event: "pull_request"},
		"should ignore non-PR events":                     {attempt: "1", event: "workflow_dispatch"},
		"should skip fork publication":                    {action: "opened", attempt: "1", event: "pull_request", fork: true},
		"should skip another PR":                          {action: "synchronize", attempt: "1", event: "pull_request", pr: 23603},
		"should skip another repository":                  {action: "synchronize", attempt: "1", event: "pull_request", repo: "someone/stackrox"},
	} {
		t.Run(name, func(t *testing.T) {
			repo, pr := test.repo, test.pr
			if repo == "" {
				repo = "stackrox/stackrox"
			}
			if pr == 0 {
				pr = 23602
			}
			context := object{"github": object{
				"event_name": test.event, "run_attempt": test.attempt, "repository": repo,
				"event": object{"action": test.action, "pull_request": object{"number": pr, "head": object{"repo": object{"fork": test.fork}}}},
			}}
			require.Equal(t, str(test.want), evaluate(t, publish["if"], context))
		})
	}
}
