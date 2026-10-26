package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecisionFromLog(t *testing.T) {
	record := "2026-10-23T12:00:00.1234567Z " + decisionMarker
	for name, test := range map[string]struct {
		log, kind, target string
		fail              bool
	}{
		"should read a selection skip":                      {log: record + selectionSkip + "\n", kind: selectionSkip},
		"should read an execution restriction":              {log: record + executionBlock + "\r\n", kind: executionBlock},
		"should read permission without claiming execution": {log: record + "permitted\n", kind: "permitted"},
		"should ignore echoed shell source":                 {log: "2026-10-23T12:00:00Z printf '" + decisionMarker + "%s\\n' \"$DECISION\"\n" + record + selectionSkip, kind: selectionSkip},
		"should reject an absent record":                    {log: "Candidate: false", fail: true},
		"should reject a truncated record":                  {log: record + "skipped-by-", fail: true},
		"should reject duplicate records":                   {log: record + selectionSkip + "\n" + record + selectionSkip, fail: true},
		"should reject conflicting records":                 {log: record + selectionSkip + "\n" + record + "permitted", fail: true},
		"should reject an unknown record":                   {log: record + "unknown", fail: true},
		"should require a GitHub timestamp":                 {log: "echo " + decisionMarker + selectionSkip, fail: true},
		"should associate a preparation with its consumer":  {log: record + selectionSkip + "\n2026-10-23T12:00:01Z " + targetMarker + "e2e-byodb-tests", kind: selectionSkip, target: "e2e-byodb-tests"},
		"should reject duplicate targets":                   {log: record + selectionSkip + "\n2026-10-23T12:00:01Z " + targetMarker + "e2e-byodb-tests\n2026-10-23T12:00:02Z " + targetMarker + "e2e-qa-tests-gke", fail: true},
		"should reject a target containing a job hierarchy": {log: record + selectionSkip + "\n2026-10-23T12:00:01Z " + targetMarker + "e2e / run", fail: true},
		"should reject a target without a decision":         {log: "2026-10-23T12:00:01Z " + targetMarker + "e2e-byodb-tests", fail: true},
	} {
		t.Run(name, func(t *testing.T) {
			decision, err := decisionFromLog([]byte(test.log))
			if test.fail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.kind, decision.kind)
				assert.Equal(t, test.target, decision.target)
			}
		})
	}
}

func TestCollectDecisions(t *testing.T) {
	for name, test := range map[string]struct {
		conclusion, step, record, kind, reason string
		readError                              bool
		requests, recorded, unattributed       int
	}{
		"should list a selection skip":                        {step: "success", record: selectionSkip, kind: selectionSkip, requests: 1, recorded: 1},
		"should list an execution block":                      {step: "success", record: executionBlock, kind: executionBlock, requests: 1, recorded: 1},
		"should omit permitted work from the exception lists": {step: "success", record: "permitted", requests: 1, recorded: 1},
		"should separate uninstrumented skips":                {conclusion: "skipped", unattributed: 1},
		"should report an instrumented job that skipped":      {conclusion: "skipped", step: "skipped", kind: noDecision, reason: "job skipped; selection explanation did not run"},
		"should report a failed explanation":                  {step: "failure", kind: noDecision, reason: "explanation did not complete successfully"},
		"should report a skipped explanation":                 {step: "skipped", kind: noDecision, reason: "explanation did not complete successfully"},
		"should retain timing when logs cannot be read":       {step: "success", readError: true, kind: noDecision, reason: "decision log unavailable", requests: 1},
		"should not parse human explanations":                 {step: "success", kind: noDecision, reason: "decision record missing from log excerpt", requests: 1},
		"should reject unsupported records":                   {step: "success", record: "unsupported", kind: noDecision, reason: "invalid decision record", requests: 1},
		"should omit jobs outside selection instrumentation":  {},
		"should omit canceled jobs":                           {conclusion: "cancelled", step: "success"},
	} {
		t.Run(name, func(t *testing.T) {
			job := workflowJob{ID: 42, Name: "example", Conclusion: test.conclusion}
			if test.step != "" {
				job.Steps = []workflowStep{{Name: "Explain selection", Conclusion: test.step}}
			}
			requests := 0
			request := func(_ context.Context, method, path string, _ []byte) ([]byte, error) {
				requests++
				assert.Equal(t, "GET", method)
				assert.Equal(t, testBase+"/actions/jobs/42/logs", path)
				if test.readError {
					return nil, errors.New("unavailable")
				}
				if test.record == "" {
					return []byte("Candidate: false\nEnforced: true"), nil
				}
				return []byte("2026-10-23T12:00:00Z " + decisionMarker + test.record), nil
			}
			report := collectDecisions(t.Context(), request, testBase, 1, []workflowJob{job, {Name: "CI timing report / publish"}})
			assert.Equal(t, int64(1), report.RunID)
			assert.Equal(t, test.requests, requests)
			assert.Equal(t, test.recorded, report.Recorded)
			assert.Equal(t, test.unattributed, report.UnattributedSkips)
			if test.kind == "" {
				assert.Empty(t, report.Jobs)
			} else {
				require.Len(t, report.Jobs, 1)
				assert.Equal(t, jobDecision{ID: 42, Name: "example", Kind: test.kind, Reason: test.reason}, report.Jobs[0])
			}
		})
	}
}

func TestCollectPreparationDecisions(t *testing.T) {
	for name, test := range map[string]struct {
		kind, conclusion string
		unattributed     int
	}{
		"should explain a skipped reusable call":     {kind: selectionSkip, conclusion: "skipped"},
		"should explain nested skipped tests":        {kind: executionBlock, conclusion: "skipped"},
		"should not infer execution from permission": {kind: "permitted", conclusion: "skipped", unattributed: 1},
		"should exclude canceled consumers":          {kind: "permitted", conclusion: "cancelled"},
	} {
		t.Run(name, func(t *testing.T) {
			request := func(_ context.Context, _, _ string, _ []byte) ([]byte, error) {
				return []byte("2026-10-23T12:00:00Z " + decisionMarker + test.kind + "\n2026-10-23T12:00:01Z " + targetMarker + "e2e-byodb-tests"), nil
			}
			jobs := []workflowJob{
				{ID: 9, Name: "e2e-byodb-tests / byodb-e2e-tests", Conclusion: test.conclusion},
				{ID: 10, Name: "select-e2e-byodb-tests", Conclusion: "success", Steps: []workflowStep{{Name: "Explain selection", Conclusion: "success"}}},
				{ID: 11, Name: "Post failure message to Slack", Conclusion: "skipped"},
				{ID: 12, Name: "e2e-byodb-tests-extra", Conclusion: "skipped"},
			}
			report := collectDecisions(t.Context(), request, testBase, 1, jobs)
			assert.Equal(t, 1, report.Recorded)
			assert.Equal(t, 2+test.unattributed, report.UnattributedSkips)
			if test.kind == "permitted" {
				assert.Empty(t, report.Jobs)
			} else {
				require.Len(t, report.Jobs, 1)
				assert.Equal(t, jobDecision{ID: 10, Name: "e2e-byodb-tests", Kind: test.kind, Reason: "recorded by preparation job"}, report.Jobs[0])
			}
		})
	}
}

func TestRenderDecisions(t *testing.T) {
	start := time.Date(2026, time.October, 23, 12, 0, 0, 0, time.UTC)
	first := measurement{SHA: testSHA, Start: start, End: start.Add(time.Minute), Reports: []selectionReport{{
		RunID: 1, Recorded: 2, Jobs: []jobDecision{{ID: 10, Name: "Go", Kind: selectionSkip}, {ID: 11, Name: "Konflux", Kind: executionBlock}},
	}}}
	second := measurement{SHA: testSHA, Start: start, End: start.Add(2 * time.Minute), Reports: []selectionReport{{
		RunID: 2, UnattributedSkips: 28, Jobs: []jobDecision{{ID: 20, Name: "E2E [untrusted] <tag>\nname", Kind: noDecision, Reason: "decision log unavailable"}},
	}}}
	body, err := render("stackrox/stackrox", "", first)
	require.NoError(t, err)
	merged, err := render("stackrox/stackrox", body, second)
	require.NoError(t, err)
	assert.Contains(t, merged, "**Skipped by selection (1)**")
	assert.Contains(t, merged, "**Blocked by execution (1)**")
	assert.Contains(t, merged, "**Explanation step present, decision unavailable (1)**")
	assert.Contains(t, merged, "**0 decisions permitted work.**")
	assert.Contains(t, merged, "28 additional jobs were skipped without selection records")
	assert.Contains(t, merged, "[Go](https://github.com/stackrox/stackrox/actions/runs/1/job/10)")
	assert.Contains(t, merged, `E2E \[untrusted\] &lt;tag&gt; name`)
	assert.Contains(t, merged, "2m0s")
	assert.Contains(t, merged, "do not estimate time saved")
	other, err := render("stackrox/stackrox", "", second)
	require.NoError(t, err)
	reversed, err := render("stackrox/stackrox", other, first)
	require.NoError(t, err)
	assert.Equal(t, merged, reversed)
	unchanged, err := render("stackrox/stackrox", merged, second)
	require.NoError(t, err)
	assert.Equal(t, merged, unchanged)
	second.Reports[0].Jobs = nil
	second.Reports[0].Recorded = 1
	recovered, err := render("stackrox/stackrox", merged, second)
	require.NoError(t, err)
	assert.NotContains(t, recovered, "untrusted", "replace the workflow contribution rather than retaining stale unknown results")
	assert.Contains(t, recovered, "**Explanation step present, decision unavailable (0)**")
	assert.Contains(t, recovered, "**1 decisions permitted work.**")
	legacy := commentMarker + "\n" + dataMarker + `[{"sha":"` + strings.Repeat("b", 40) + `","start":"2026-10-22T12:00:00Z","end":"2026-10-22T12:10:00Z"}] -->`
	history, err := render("stackrox/stackrox", legacy, first)
	require.NoError(t, err)
	assert.Contains(t, history, "No decision summary was recorded for this measurement.")
	assert.Contains(t, history, strings.Repeat("b", 40))
	first.Reports[0].Jobs = append(first.Reports[0].Jobs, jobDecision{ID: 30, Name: "Slack notification", Kind: noDecision, Reason: "job skipped before recording a decision"})
	oldData, err := render("stackrox/stackrox", "", first)
	require.NoError(t, err)
	visible, _, _ := strings.Cut(oldData, dataMarker)
	assert.NotContains(t, visible, "Slack notification")
	assert.Contains(t, visible, "1 additional jobs were skipped without selection records")
}
