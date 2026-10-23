package main

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMeasure(t *testing.T) {
	start := time.Date(2026, time.October, 23, 12, 0, 0, 0, time.UTC)
	job := func(name, conclusion string, duration time.Duration) workflowJob {
		return workflowJob{Name: name, Status: "completed", Conclusion: conclusion, CompletedAt: start.Add(duration)}
	}
	for name, test := range map[string]struct {
		jobs []workflowJob
		wait time.Duration
	}{
		"should measure parallel work once including queues and dependency gaps": {
			jobs: []workflowJob{job("fast", "success", 10*time.Minute), job("slow", "success", 15*time.Minute)}, wait: 15 * time.Minute,
		},
		"should include a failed check finishing last": {
			jobs: []workflowJob{job("passed", "success", time.Minute), job("failed", "failure", 8*time.Minute)}, wait: 8 * time.Minute,
		},
		"should include successful green skips": {
			jobs: []workflowJob{job("test", "success", time.Minute), job("green no-op", "success", 3*time.Minute)}, wait: 3 * time.Minute,
		},
		"should include conditionally skipped checks without a runner": {
			jobs: []workflowJob{job("test", "success", time.Minute), job("skipped", "skipped", 2*time.Minute)}, wait: 2 * time.Minute,
		},
		"should exclude canceled checks": {
			jobs: []workflowJob{job("failed", "failure", time.Minute), job("cancelled", "cancelled", 20*time.Minute)}, wait: time.Minute,
		},
		"should exclude active and completed reporters": {
			jobs: []workflowJob{
				job("test", "success", time.Minute),
				job("CI timing report / publish", "success", 20*time.Minute),
				{Name: "CI timing report", Status: "in_progress"},
				{Name: "Build / CI timing report / publish", Status: "in_progress"},
			}, wait: time.Minute,
		},
	} {
		t.Run(name, func(t *testing.T) {
			result, err := measure(workflowRun{HeadSHA: testSHA, CreatedAt: start}, test.jobs)
			require.NoError(t, err)
			assert.Equal(t, start, result.Start)
			assert.Equal(t, start.Add(test.wait), result.End)
			assert.Equal(t, test.wait, result.End.Sub(result.Start))
		})
	}
	for name, jobs := range map[string][]workflowJob{
		"should reject a running check":                           {{Name: "running", Status: "in_progress"}},
		"should reject missing completion timestamps":             {{Name: "missing", Status: "completed"}},
		"should reject an empty measurement":                      {},
		"should reject checks finishing before workflow creation": {job("invalid", "success", -time.Minute)},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := measure(workflowRun{HeadSHA: testSHA, CreatedAt: start}, jobs)
			require.Error(t, err)
		})
	}
}

func TestRender(t *testing.T) {
	start := time.Date(2026, time.October, 23, 12, 0, 0, 0, time.UTC)
	first := measurement{SHA: testSHA, Start: start, End: start.Add(10 * time.Minute)}
	later := measurement{SHA: testSHA, Start: start.Add(5 * time.Minute), End: start.Add(15 * time.Minute)}
	a, err := render("stackrox/stackrox", "", first)
	require.NoError(t, err)
	merged, err := render("stackrox/stackrox", a, later)
	require.NoError(t, err)
	assert.Contains(t, merged, "| 2026-10-23 12:00:00 | 2026-10-23 12:15:00 | 15m0s |")
	assert.Equal(t, 1, strings.Count(merged, "/commit/"+testSHA), "one row per commit")
	b, err := render("stackrox/stackrox", "", later)
	require.NoError(t, err)
	reversed, err := render("stackrox/stackrox", b, first)
	require.NoError(t, err)
	assert.Equal(t, merged, reversed, "workflow completion and publication order must not affect the result")
	unchanged, err := render("stackrox/stackrox", merged, later)
	require.NoError(t, err)
	assert.Equal(t, merged, unchanged, "repeated contributions must not edit the comment")
	other := measurement{SHA: strings.Repeat("b", 40), Start: start.Add(24 * time.Hour), End: start.Add(24*time.Hour + time.Minute)}
	history, err := render("stackrox/stackrox", merged, other)
	require.NoError(t, err)
	assert.Contains(t, history, "/commit/"+testSHA)
	assert.Contains(t, history, "/commit/"+other.SHA)
	assert.Contains(t, history, "provisional")
	assert.Contains(t, history, dataMarker)
}

func TestRenderInvalidHistory(t *testing.T) {
	current := measurement{SHA: testSHA, Start: time.Now(), End: time.Now().Add(time.Minute)}
	for name, body := range map[string]string{
		"should reject missing hidden data":      commentMarker,
		"should reject unterminated hidden data": commentMarker + dataMarker + "[]",
		"should reject malformed hidden JSON":    commentMarker + dataMarker + "{ -->",
		"should reject invalid timestamps":       commentMarker + dataMarker + `[{"sha":"` + testSHA + `"}] -->`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := render("stackrox/stackrox", body, current)
			require.Error(t, err)
		})
	}
}
