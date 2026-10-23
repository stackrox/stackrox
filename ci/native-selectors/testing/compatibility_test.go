package testing_test

import (
	"testing"

	"github.com/nektos/act/pkg/exprparser"
	"github.com/nektos/act/pkg/model"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestEmptyLabelCompatibility records act's empty-filter mismatch without normalizing its output.
func TestEmptyLabelCompatibility(t *testing.T) {
	for name, event := range map[string]object{
		"empty PR labels":   {"pull_request": object{"labels": []any{}}},
		"missing PR labels": {"pull_request": object{}},
		"non-PR event":      {},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := object{"github": object{"event": event}, "steps": object{"labels": object{"outputs": object{}}}}
			require.Equal(t, "null", evaluate(t, "toJSON(github.event.pull_request.labels.*.name)", ctx))
			require.Equal(t, "false", evaluate(t, "toJSON(github.event.pull_request.labels.*.name) == '[]'", ctx))
			require.Equal(t, "null", evaluate(t, "steps.labels.outputs.labels || toJSON(github.event.pull_request.labels.*.name) || '[]'", ctx))
			require.Equal(t, "false", evaluate(t, "contains(fromJSON(steps.labels.outputs.labels || toJSON(github.event.pull_request.labels.*.name) || '[]'), 'ci-dispatcher-enforce')", ctx))
			t.Log("GitHub @actions/expressions 0.3.0 returns [] and true for the first two expressions; see README.md")
		})
	}
}

func TestStatusAndPrerequisites(t *testing.T) {
	tests := map[string]struct {
		source, scope, status, parent string
		defaultStatus                 exprparser.DefaultStatusCheck
		want                          bool
	}{
		"step success":                                       {source: "success()", scope: "step", status: "success", want: true},
		"step failure":                                       {source: "failure()", scope: "step", status: "failure", want: true},
		"step success excludes failure":                      {source: "success()", scope: "step", status: "failure"},
		"step cancellation":                                  {source: "cancelled()", scope: "step", status: "cancelled", want: true},
		"not cancelled excludes cancellation":                {source: "!cancelled()", scope: "step", status: "cancelled"},
		"always survives failure":                            {source: "always()", scope: "step", status: "failure", want: true},
		"default step success":                               {source: "true", scope: "step", status: "success", defaultStatus: exprparser.DefaultStatusCheckSuccess, want: true},
		"default step success excludes failure":              {source: "true", scope: "step", status: "failure", defaultStatus: exprparser.DefaultStatusCheckSuccess},
		"explicit always bypasses default":                   {source: "always()", scope: "step", status: "failure", defaultStatus: exprparser.DefaultStatusCheckSuccess, want: true},
		"job successful image":                               {source: "success()", scope: "job", status: "success", parent: "success", want: true},
		"job failed image":                                   {source: "success()", scope: "job", status: "success", parent: "failure"},
		"job skipped image":                                  {source: "success()", scope: "job", status: "success", parent: "skipped"},
		"job cancelled image":                                {source: "success()", scope: "job", status: "success", parent: "cancelled"},
		"job failure detects failed image":                   {source: "failure()", scope: "job", status: "success", parent: "failure", want: true},
		"default job condition rejects failed image":         {source: "true", scope: "job", status: "success", parent: "failure", defaultStatus: exprparser.DefaultStatusCheckSuccess},
		"explicit not cancelled permits failed prerequisite": {source: "!cancelled()", scope: "job", status: "success", parent: "failure", defaultStatus: exprparser.DefaultStatusCheckSuccess, want: true},
		"invalid JSON is not used in short circuit":          {source: "false && fromJSON('broken')", scope: "step", status: "success"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			workflow := &model.Workflow{Jobs: map[string]*model.Job{
				"image": {Result: test.parent}, "consumer": {RawNeeds: yaml.Node{Kind: yaml.ScalarNode, Value: "image"}},
			}}
			interpreter := exprparser.NewInterpeter(&exprparser.EvaluationEnvironment{Job: &model.JobContext{Status: test.status}}, exprparser.Config{Context: test.scope, Run: &model.Run{Workflow: workflow, JobID: "consumer"}})
			got, err := interpreter.Evaluate(test.source, test.defaultStatus)
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}
