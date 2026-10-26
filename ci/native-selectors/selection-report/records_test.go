package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadRecord(t *testing.T) {
	for name, test := range map[string]struct {
		text, status, conclusion, jobStatus, want string
		explain, recorded                         bool
	}{
		"should read permission before tests finish":   {text: "permitted | go | candidate=true | eligible=true | enforced=true", status: "completed", conclusion: "success", want: "permitted", recorded: true},
		"should read an enforced selection skip":       {text: "skipped-by-selection | ui | candidate=false | eligible=true | enforced=true", status: "completed", conclusion: "success", want: "skipped-by-selection", recorded: true},
		"should read a blocked E2E call":               {text: "blocked-by-execution | e2e-qa-tests-gke-konflux | candidate=true | eligible=false | enforced=true", status: "completed", conclusion: "success", want: "blocked-by-execution", recorded: true},
		"should retain shadow permission":              {text: "permitted | go | candidate=false | eligible=true | enforced=false", status: "completed", conclusion: "success", want: "permitted", recorded: true},
		"should wait for a metadata step to finish":    {text: "permitted | go | candidate=true | eligible=true | enforced=true", status: "in_progress", want: "pending", recorded: true},
		"should expose a failed explanation":           {text: "unavailable | go | candidate= | eligible= | enforced=", status: "completed", conclusion: "success", want: "unavailable", recorded: true},
		"should reject invalid flags":                  {text: "permitted | go | candidate=maybe | eligible=true | enforced=true", status: "completed", conclusion: "success", want: "unavailable", recorded: true},
		"should reject an unsupported result":          {text: "unknown | go | candidate=true | eligible=true | enforced=true", status: "completed", conclusion: "success", want: "unavailable", recorded: true},
		"should expose a job ending before its record": {explain: true, jobStatus: "completed", want: "unavailable", recorded: true},
		"should wait for an explanation":               {explain: true, want: "pending", recorded: true},
		"should omit unrelated skipped jobs":           {jobStatus: "completed"},
	} {
		t.Run(name, func(t *testing.T) {
			job := workflowJob{ID: 10, Name: "go", Status: test.jobStatus}
			if job.Status == "" {
				job.Status = "in_progress"
			}
			if test.explain {
				job.Steps = append(job.Steps, workflowStep{Name: "Explain selection"})
			}
			if test.text != "" {
				job.Steps = append(job.Steps, workflowStep{Name: recordPrefix + test.text, Status: test.status, Conclusion: test.conclusion})
			}
			row, recorded := readRecord(workflowRun{ID: 1, Name: "Unit Tests"}, job)
			assert.Equal(t, test.recorded, recorded)
			if recorded {
				assert.Equal(t, test.want, row.kind)
				assert.Equal(t, int64(10), row.id)
			}
		})
	}
}

func TestRender(t *testing.T) {
	opt := options{repo: "stackrox/stackrox", sha: testSHA, mode: "Enforce", run: 99, job: 100, interval: 5 * time.Second, watch: time.Minute}
	deadline := time.Date(2026, time.October, 26, 12, 0, 0, 0, time.UTC)
	for name, test := range map[string]struct {
		rows   []record
		active bool
		mode   string
		want   string
	}{
		"should state waiting rather than report zero skips":        {active: true, want: "Waiting for native decision records"},
		"should show a native rejection":                            {rows: []record{{id: 10, run: 1, workflow: "Unit Tests", name: "go", kind: "skipped-by-selection", candidate: "false", eligible: "true", enforced: "true"}}, active: true, want: "| Skip |"},
		"should distinguish permission from execution":              {rows: []record{{id: 10, run: 1, name: "go", kind: "permitted"}}, active: true, want: "These are selection decisions, not test results"},
		"should show selection despite a blocked execution":         {rows: []record{{id: 10, run: 1, name: "konflux", kind: "blocked-by-execution", candidate: "true", eligible: "false", enforced: "true"}}, want: "| Allow |"},
		"should show a skip recommendation in shadow mode":          {rows: []record{{id: 10, run: 1, name: "go", kind: "permitted", candidate: "false", eligible: "true", enforced: "false"}}, mode: "Shadow", want: "| Skip |"},
		"should keep the requested mode despite advisory jobs":      {rows: []record{{id: 10, run: 1, name: "image-report", kind: "permitted", candidate: "true", eligible: "true", enforced: "false"}}, want: "| Allow |"},
		"should state that observation ended":                       {want: "Observation window ended"},
		"should escape names rather than generate Markdown or HTML": {rows: []record{{id: 10, run: 1, workflow: "<script>", name: "E2E [a]|b\nnext", kind: "unavailable"}}, want: `E2E \[a\]\|b next`},
	} {
		t.Run(name, func(t *testing.T) {
			current := opt
			if test.mode != "" {
				current.mode = test.mode
			}
			body := render(current, test.rows, test.active, deadline)
			assert.Contains(t, body, test.want)
			assert.Contains(t, body, "https://github.com/stackrox/stackrox/actions/runs/99/job/100")
			assert.NotContains(t, body, "<script>")
			assert.NotContains(t, body, "Skip (0)")
			assert.NotContains(t, body, "ci-timing-data:")
			assert.Contains(t, body, "**Mode: "+current.mode+"**")
			assert.NotContains(t, body, "| Mode |")
			assert.NotContains(t, body, "| Other run rules |")
			assert.NotContains(t, body, "| Blocked |")
		})
	}
	var rows []record
	for i := range 500 {
		rows = append(rows, record{id: int64(i + 1), run: 1, name: strings.Repeat(fmt.Sprintf("job-%d ", i), 100), kind: "permitted"})
	}
	body := render(opt, rows, true, deadline)
	assert.Less(t, len(body), 60000)
	assert.Contains(t, body, "additional records omitted")
	require.Contains(t, body, commentMarker)
}
