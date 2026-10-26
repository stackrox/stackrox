package main

import (
	"encoding/json"
	"fmt"
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

func TestRenderHistoryRetention(t *testing.T) {
	start := time.Date(2026, time.October, 23, 0, 0, 0, 0, time.UTC)
	var rows []measurement
	for i := 1; i <= 7; i++ {
		rows = append(rows, measurement{
			SHA: fmt.Sprintf("%040x", i), Start: start.Add(time.Duration(i) * time.Hour), End: start.Add(time.Duration(i)*time.Hour + time.Minute),
			Reports: []selectionReport{{RunID: int64(i), Recorded: 1, Jobs: []jobDecision{{ID: int64(i), Name: fmt.Sprintf("job-for-commit-%d", i), Kind: selectionSkip}}}},
		})
	}
	old, err := renderRows("stackrox/stackrox", rows[:6])
	require.NoError(t, err)
	body, err := render("stackrox/stackrox", old, rows[6])
	require.NoError(t, err)
	kept := storedMeasurements(t, body)
	require.Len(t, kept, 5)
	assert.Equal(t, rows[2:], kept)
	for _, row := range rows[:2] {
		assert.NotContains(t, body, row.SHA, "discard timestamps and serialized details together")
		assert.NotContains(t, body, row.Reports[0].Jobs[0].Name)
		late := row
		late.End = late.End.Add(4 * time.Hour)
		unchanged, err := render("stackrox/stackrox", body, late)
		require.NoError(t, err)
		assert.Equal(t, body, unchanged, "a late contribution to an evicted commit must not displace recent rows")
	}
	late := rows[3]
	late.End = late.End.Add(time.Hour)
	updated, err := render("stackrox/stackrox", body, late)
	require.NoError(t, err)
	kept = storedMeasurements(t, updated)
	require.Len(t, kept, 5)
	assert.Equal(t, late.End, kept[1].End)
	unchanged, err := render("stackrox/stackrox", updated, late)
	require.NoError(t, err)
	assert.Equal(t, updated, unchanged)
}

func TestRenderCommentByteLimit(t *testing.T) {
	start := time.Date(2026, time.October, 23, 0, 0, 0, 0, time.UTC)
	var rows []measurement
	for i := 1; i <= 5; i++ {
		rows = append(rows, measurement{
			SHA: fmt.Sprintf("%040x", i), Start: start.Add(time.Duration(i) * time.Hour), End: start.Add(time.Duration(i)*time.Hour + time.Minute),
			Reports: []selectionReport{{RunID: int64(i), Recorded: 1, Jobs: []jobDecision{{ID: int64(i), Name: strings.Repeat(fmt.Sprintf("selection-%d ", i), 1000), Kind: executionBlock}}}},
		})
	}
	oversized, err := renderRows("stackrox/stackrox", rows)
	require.NoError(t, err)
	require.Greater(t, len(oversized), maxCommentBytes, "five commits alone must not be assumed to fit")
	body, err := render("stackrox/stackrox", oversized, rows[4])
	require.NoError(t, err)
	assert.LessOrEqual(t, len(body), maxCommentBytes)
	kept := storedMeasurements(t, body)
	require.Len(t, kept, 5, "byte compaction retains the timings")
	assert.True(t, kept[0].DetailsOmitted)
	assert.False(t, kept[4].DetailsOmitted, "omit older details before newer details")
	for i, row := range kept {
		assert.Equal(t, rows[i].Start, row.Start)
		assert.Equal(t, rows[i].End, row.End)
		if row.DetailsOmitted {
			assert.Empty(t, row.Reports)
			assert.NotContains(t, body, rows[i].Reports[0].Jobs[0].Name)
		}
	}
	late := rows[0]
	late.End = late.End.Add(10 * time.Hour)
	updated, err := render("stackrox/stackrox", body, late)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(updated), maxCommentBytes)
	kept = storedMeasurements(t, updated)
	assert.Equal(t, late.End, kept[0].End)
	assert.True(t, kept[0].DetailsOmitted)
	assert.Empty(t, kept[0].Reports, "late contributions must not regrow compacted detail history")
}

// storedMeasurements checks the hidden data as well as the visible Markdown.
func storedMeasurements(t *testing.T, body string) []measurement {
	t.Helper()
	_, encoded, ok := strings.Cut(body, dataMarker)
	require.True(t, ok)
	encoded, _, ok = strings.Cut(encoded, " -->")
	require.True(t, ok)
	var rows []measurement
	require.NoError(t, json.Unmarshal([]byte(encoded), &rows))
	return rows
}
