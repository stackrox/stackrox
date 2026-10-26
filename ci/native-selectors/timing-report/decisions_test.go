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
		log, kind string
		fail      bool
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
	} {
		t.Run(name, func(t *testing.T) {
			kind, err := decisionFromLog([]byte(test.log))
			if test.fail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.kind, kind)
			}
		})
	}
}

func TestCollectDecisions(t *testing.T) {
	for name, test := range map[string]struct {
		conclusion, step, record, kind, reason string
		readError                              bool
		requests, recorded                     int
	}{
		"should list a selection skip":                        {step: "success", record: selectionSkip, kind: selectionSkip, requests: 1, recorded: 1},
		"should list an execution block":                      {step: "success", record: executionBlock, kind: executionBlock, requests: 1, recorded: 1},
		"should omit permitted work from the exception lists": {step: "success", record: "permitted", requests: 1, recorded: 1},
		"should not infer why a whole job skipped":            {conclusion: "skipped", kind: noDecision, reason: "job skipped before recording a decision"},
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
			if test.kind == "" {
				assert.Empty(t, report.Jobs)
			} else {
				require.Len(t, report.Jobs, 1)
				assert.Equal(t, jobDecision{ID: 42, Name: "example", Kind: test.kind, Reason: test.reason}, report.Jobs[0])
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
		RunID: 2, Jobs: []jobDecision{{ID: 20, Name: "E2E [untrusted] <tag>\nname", Kind: noDecision, Reason: "job skipped before recording a decision"}},
	}}}
	body, err := render("stackrox/stackrox", "", first)
	require.NoError(t, err)
	merged, err := render("stackrox/stackrox", body, second)
	require.NoError(t, err)
	assert.Contains(t, merged, "**Skipped by selection (1)**")
	assert.Contains(t, merged, "**Blocked by execution (1)**")
	assert.Contains(t, merged, "**No recorded decision (1)**")
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
	assert.Contains(t, recovered, "**No recorded decision (0)**")
	legacy := commentMarker + "\n" + dataMarker + `[{"sha":"` + strings.Repeat("b", 40) + `","start":"2026-10-22T12:00:00Z","end":"2026-10-22T12:10:00Z"}] -->`
	history, err := render("stackrox/stackrox", legacy, first)
	require.NoError(t, err)
	assert.Contains(t, history, "No decision summary was recorded for this measurement.")
	assert.Contains(t, history, strings.Repeat("b", 40))
}
