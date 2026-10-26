package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testSHA    = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	testRepo   = "stackrox/stackrox"
	testPrefix = "repos/" + testRepo
)

type fakeGitHub struct {
	data          map[string]any
	calls, writes []string
	bodies        []string
}

func newFake() *fakeGitHub {
	f := &fakeGitHub{data: map[string]any{}}
	f.data[testPrefix+"/pulls/23602"] = map[string]any{"head": map[string]any{"sha": testSHA, "ref": "piotr/prototype", "repo": map[string]any{"full_name": testRepo}}}
	f.data[runPath(testRepo, testSHA, 1)] = map[string]any{"workflow_runs": []workflowRun{{ID: 1, WorkflowID: 10, Name: "Unit Tests", HeadSHA: testSHA, PRs: []struct{ Number int }{{23602}}}}}
	f.data[testPrefix+"/actions/runs/1/jobs?filter=all&per_page=100&page=1"] = map[string]any{"jobs": []workflowJob{{ID: 11, Name: "go", Status: "in_progress", Attempt: 1, Steps: []workflowStep{{Name: "Explain selection", Status: "completed", Conclusion: "success"}, {Name: recordPrefix + "permitted | go | candidate=true | eligible=true | enforced=true", Status: "completed", Conclusion: "success"}}}}}
	f.data[testPrefix+"/issues/23602/comments?per_page=100&page=1"] = []comment{}
	return f
}

func (f *fakeGitHub) request(_ context.Context, method, path string, body []byte, _ string) (response, error) {
	f.calls = append(f.calls, method+" "+path)
	if method != "GET" {
		var data map[string]string
		if err := json.Unmarshal(body, &data); err != nil {
			return response{}, err
		}
		f.writes = append(f.writes, method+" "+path)
		f.bodies = append(f.bodies, data["body"])
		return response{status: 200, body: []byte(`{"id":42,"user":{"login":"github-actions[bot]"}}`)}, nil
	}
	value, ok := f.data[path]
	if !ok {
		return response{}, fmt.Errorf("unexpected request %s", path)
	}
	encoded, err := json.Marshal(value)
	return response{status: 200, etag: "version-1", body: encoded}, err
}

func TestObservePublishesBeforeTestsFinish(t *testing.T) {
	f := newFake()
	now := time.Date(2026, time.October, 26, 12, 0, 0, 0, time.UTC)
	start := now
	clock := clock{now: func() time.Time { return now }, sleep: func(_ context.Context, d time.Duration) error { now = now.Add(d); return nil }}
	var output, log bytes.Buffer
	err := observe(t.Context(), &api{request: f.request, cache: map[string]response{}}, options{repo: testRepo, pr: 23602, sha: testSHA, watch: 10 * time.Second, interval: 5 * time.Second, post: true}, clock, &output, &log)
	require.NoError(t, err)
	require.Len(t, f.writes, 2, "publish immediately, then mark the bounded window ended; no duplicate edits")
	assert.Contains(t, f.bodies[0], "Work permitted")
	assert.Contains(t, f.bodies[0], "Watching for decisions")
	assert.Contains(t, f.bodies[1], "Observation window ended")
	assert.Equal(t, start.Add(10*time.Second), now)
	assert.Contains(t, log.String(), "go | permitted")
	for _, call := range f.calls {
		assert.NotContains(t, call, "/logs", "running-job log downloads must not be needed")
	}
}

func TestObservePublishesANewDecisionWhileTestsRun(t *testing.T) {
	f := newFake()
	jobsPath := testPrefix + "/actions/runs/1/jobs?filter=all&per_page=100&page=1"
	job := workflowJob{ID: 11, Name: "go (1.26)", Status: "in_progress", Steps: []workflowStep{{Name: "Explain selection", Status: "in_progress"}}}
	f.data[jobsPath] = map[string]any{"jobs": []workflowJob{job}}
	now := time.Date(2026, time.October, 26, 12, 0, 0, 0, time.UTC)
	c := clock{now: func() time.Time { return now }, sleep: func(_ context.Context, duration time.Duration) error {
		now = now.Add(duration)
		if job.Steps[0].Name == "Explain selection" {
			job.Steps = []workflowStep{{Name: recordPrefix + "skipped-by-selection | go | candidate=false | eligible=true | enforced=true", Status: "completed", Conclusion: "success", CompletedAt: now}}
			f.data[jobsPath] = map[string]any{"jobs": []workflowJob{job}}
		}
		return nil
	}}
	var output, log bytes.Buffer
	require.NoError(t, observe(t.Context(), &api{request: f.request, cache: map[string]response{}}, options{repo: testRepo, pr: 23602, sha: testSHA, watch: 15 * time.Second, interval: 5 * time.Second, post: true}, c, &output, &log))
	require.Len(t, f.bodies, 3, "pending, recorded, then window ended; unchanged observations do not write")
	assert.Contains(t, f.bodies[0], "Decision pending")
	assert.Contains(t, f.bodies[1], "Skipped by selection")
	assert.Contains(t, f.bodies[2], "Observation window ended")
	assert.Equal(t, "in_progress", job.Status, "publication must not wait for test completion")
	assert.Contains(t, log.String(), "source=go (1.26)")
}

func TestCollectScopesAndReruns(t *testing.T) {
	f := newFake()
	f.data[runPath(testRepo, testSHA, 1)] = map[string]any{"workflow_runs": []workflowRun{
		{ID: 1, WorkflowID: 10, Name: "Unit Tests", HeadSHA: testSHA, CreatedAt: time.Date(2026, time.October, 26, 12, 0, 0, 0, time.UTC), PRs: []struct{ Number int }{{23602}}},
		{ID: 2, WorkflowID: 10, Name: "Unit Tests", HeadSHA: testSHA, CreatedAt: time.Date(2026, time.October, 26, 12, 1, 0, 0, time.UTC), PRs: []struct{ Number int }{{23602}}},
		{ID: 3, WorkflowID: 20, HeadSHA: testSHA, PRs: []struct{ Number int }{{123}}},
	}}
	f.data[testPrefix+"/actions/runs/2/jobs?filter=all&per_page=100&page=1"] = map[string]any{"jobs": []workflowJob{
		{ID: 20, Name: "go", Attempt: 1, Steps: []workflowStep{{Name: recordPrefix + "permitted | go | candidate=true | eligible=true | enforced=true", Status: "completed", Conclusion: "success"}}},
		{ID: 21, Name: "go", Attempt: 2, Steps: []workflowStep{{Name: recordPrefix + "skipped-by-selection | go | candidate=false | eligible=true | enforced=true", Status: "completed", Conclusion: "success"}}},
		{ID: 22, Name: "go-postgres", Attempt: 1, Steps: []workflowStep{{Name: recordPrefix + "permitted | go-postgres | candidate=true | eligible=true | enforced=true", Status: "completed", Conclusion: "success"}}},
		{ID: 23, Name: "go (1.26)", Attempt: 1, Steps: []workflowStep{{Name: recordPrefix + "permitted | go | candidate=true | eligible=true | enforced=true", Status: "completed", Conclusion: "success"}}},
	}}
	rows, err := collect(t.Context(), &api{request: f.request, cache: map[string]response{}}, options{repo: testRepo, pr: 23602, sha: testSHA})
	require.NoError(t, err)
	require.Len(t, rows, 3)
	assert.Equal(t, "skipped-by-selection", rows[0].kind)
	assert.Equal(t, int64(21), rows[0].id)
	assert.Equal(t, int64(23), rows[1].id, "matrix jobs retain separate records even with the same logical target")
	assert.Equal(t, int64(22), rows[2].id, "partial reruns retain jobs not rerun")
	for _, call := range f.calls {
		assert.NotContains(t, call, "/runs/1/jobs")
		assert.NotContains(t, call, "/runs/3/jobs")
	}
}

func TestCollectAssociationFallback(t *testing.T) {
	for name, test := range map[string]struct {
		branch, repo string
		want         bool
	}{
		"should accept the same source when associations are omitted": {branch: "piotr/prototype", repo: testRepo, want: true},
		"should reject a different branch":                            {branch: "another", repo: testRepo},
		"should reject a different repository":                        {branch: "piotr/prototype", repo: "fork/stackrox"},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFake()
			run := workflowRun{ID: 1, WorkflowID: 10, Name: "Unit Tests", HeadSHA: testSHA, HeadBranch: test.branch}
			run.HeadRepo.FullName = test.repo
			f.data[runPath(testRepo, testSHA, 1)] = map[string]any{"workflow_runs": []workflowRun{run}}
			rows, err := collect(t.Context(), &api{request: f.request, cache: map[string]response{}}, options{repo: testRepo, pr: 23602, sha: testSHA})
			require.NoError(t, err)
			assert.Equal(t, test.want, len(rows) == 1)
		})
	}
}

func TestPublishOwnership(t *testing.T) {
	for name, test := range map[string]struct{ owner, method string }{
		"should reuse the marked bot comment":             {owner: "github-actions[bot]", method: "PATCH"},
		"should leave a human's marked comment untouched": {owner: "maintainer", method: "POST"},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFake()
			item := comment{ID: 42, Body: commentMarker + "\nold timing data"}
			item.User.Login = test.owner
			f.data[testPrefix+"/issues/23602/comments?per_page=100&page=1"] = []comment{item}
			var saved comment
			require.NoError(t, publish(t.Context(), &api{request: f.request, cache: map[string]response{}}, options{repo: testRepo, pr: 23602}, &saved, commentMarker+"\nselection only"))
			require.Len(t, f.writes, 1)
			assert.True(t, strings.HasPrefix(f.writes[0], test.method))
		})
	}
}

func TestPublicationGuardsAndStaleHeads(t *testing.T) {
	f := newFake()
	var out, log bytes.Buffer
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("GITHUB_TOKEN", "")
	err := run(t.Context(), []string{"--pr", "23602", "--sha", testSHA, "--post"}, f.request, &out, &log)
	require.ErrorContains(t, err, "local commands preview only")
	assert.Empty(t, f.calls)
	f.data[testPrefix+"/pulls/23602"] = map[string]any{"head": map[string]any{"sha": strings.Repeat("b", 40)}}
	_, err = collect(t.Context(), &api{request: f.request, cache: map[string]response{}}, options{repo: testRepo, pr: 23602, sha: testSHA})
	require.ErrorIs(t, err, errNewHead)
	assert.Empty(t, f.writes)
}

func TestConditionalRequests(t *testing.T) {
	calls := 0
	a := &api{cache: map[string]response{}, request: func(_ context.Context, _, _ string, _ []byte, etag string) (response, error) {
		calls++
		if calls == 1 {
			assert.Empty(t, etag)
			return response{status: 200, etag: "v1", body: []byte(`{"value":1}`)}, nil
		}
		assert.Equal(t, "v1", etag)
		return response{status: 304}, nil
	}}
	for range 2 {
		var value map[string]int
		require.NoError(t, a.get(t.Context(), "example", &value))
		assert.Equal(t, 1, value["value"])
	}
}

func TestGHRequestHeaders(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "gh"), []byte("#!/bin/sh\nprintf 'HTTP/2.0 200 OK\\r\\nETag: v1\\r\\n\\r\\n{\"ok\":true}'\n"), 0o700))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	result, err := ghRequest(t.Context(), "GET", "example", nil, "")
	require.NoError(t, err)
	assert.Equal(t, 200, result.status)
	assert.Equal(t, "v1", result.etag)
	assert.JSONEq(t, `{"ok":true}`, string(result.body))
}

func TestObserveFailureAndCancellation(t *testing.T) {
	for name, fail := range map[string]bool{"API failure": true, "cancellation": false} {
		t.Run(name, func(t *testing.T) {
			f := newFake()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if fail {
				delete(f.data, runPath(testRepo, testSHA, 1))
			}
			c := clock{now: time.Now, sleep: func(context.Context, time.Duration) error { cancel(); return context.Canceled }}
			var out, log bytes.Buffer
			err := observe(ctx, &api{request: f.request, cache: map[string]response{}}, options{repo: testRepo, pr: 23602, sha: testSHA, watch: time.Minute, interval: 5 * time.Second}, c, &out, &log)
			if fail {
				require.ErrorContains(t, err, "unexpected request")
			} else {
				require.True(t, errors.Is(err, context.Canceled))
			}
			assert.Empty(t, f.writes)
		})
	}
}
