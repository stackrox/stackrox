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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testSHA  = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	testBase = "repos/stackrox/stackrox"
	testPR   = testBase + "/pulls/23602"
	testRun  = testBase + "/actions/runs/1/attempts/1"
)

// fakeGitHub supplies paginated REST responses and captures comment mutations.
type fakeGitHub struct {
	responses map[string]string
	writes    []string
	bodies    []string
}

func newFakeGitHub() *fakeGitHub {
	return &fakeGitHub{responses: map[string]string{
		testPR:  `[{"head":{"sha":"` + testSHA + `","ref":"piotr/prototype","repo":{"full_name":"stackrox/stackrox"}}}]`,
		testRun: `[{"run_attempt":1,"event":"pull_request","head_sha":"` + testSHA + `","created_at":"2026-10-23T12:00:00Z","pull_requests":[{"number":23602}]}]`,
		testRun + "/jobs?per_page=100": `[
			{"jobs":[{"name":"test","status":"completed","conclusion":"failure","completed_at":"2026-10-23T12:10:00Z"}]},
			{"jobs":[{"name":"green skip","status":"completed","conclusion":"success","completed_at":"2026-10-23T12:15:00Z"},{"name":"CI timing report / publish","status":"in_progress"}]}
		]`,
		testBase + "/issues/23602/comments?per_page=100": `[[]]`,
	}}
}

func (f *fakeGitHub) request(_ context.Context, method, path string, body []byte) ([]byte, error) {
	if method != "GET" {
		f.writes = append(f.writes, method+" "+path)
		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, err
		}
		f.bodies = append(f.bodies, payload["body"])
		return []byte(`{"html_url":"https://github.com/stackrox/stackrox/pull/23602#issuecomment-42"}`), nil
	}
	if response, ok := f.responses[path]; ok {
		return []byte(response), nil
	}
	return nil, fmt.Errorf("unexpected request: %s", path)
}

func TestRefresh(t *testing.T) {
	for name, test := range map[string]struct {
		post        bool
		owner, want string
	}{
		"should preview without writing":                         {},
		"should create one bot comment":                          {post: true, want: "POST " + testBase + "/issues/23602/comments"},
		"should update the bot's marked comment on a later page": {post: true, owner: "github-actions[bot]", want: "PATCH " + testBase + "/issues/comments/42"},
		"should leave a human's marked comment untouched":        {post: true, owner: "maintainer", want: "POST " + testBase + "/issues/23602/comments"},
	} {
		t.Run(name, func(t *testing.T) {
			fake := newFakeGitHub()
			comments := []issueComment{}
			if test.owner != "" {
				comment := issueComment{ID: 42, Body: commentMarker + "\n" + dataMarker + "[] -->"}
				comment.User.Login = test.owner
				comments = append(comments, comment)
			}
			encoded, err := json.Marshal([][]issueComment{{}, comments})
			require.NoError(t, err)
			fake.responses[testBase+"/issues/23602/comments?per_page=100"] = string(encoded)
			var output, diagnostic bytes.Buffer
			require.NoError(t, refresh(t.Context(), fake.request, "stackrox/stackrox", 23602, 1, test.post, &output, &diagnostic))
			body := output.String()
			if test.want == "" {
				assert.Empty(t, fake.writes)
			} else {
				require.Equal(t, []string{test.want}, fake.writes)
				body = fake.bodies[0]
				assert.Contains(t, diagnostic.String(), "#issuecomment-42")
			}
			assert.Contains(t, body, "| 2026-10-23 12:00:00 | 2026-10-23 12:15:00 | 15m0s |")
		})
	}
}

func TestRefreshDecisionSummary(t *testing.T) {
	fake := newFakeGitHub()
	fake.responses[testRun+"/jobs?per_page=100"] = `[{"jobs":[
		{"id":10,"name":"selection skip","status":"completed","conclusion":"success","completed_at":"2026-10-23T12:10:00Z","steps":[{"name":"Explain selection","conclusion":"success"}]},
		{"id":11,"name":"execution block","status":"completed","conclusion":"success","completed_at":"2026-10-23T12:12:00Z","steps":[{"name":"Explain selection","conclusion":"success"}]},
		{"id":12,"name":"skipped E2E caller","status":"completed","conclusion":"skipped","completed_at":"2026-10-23T12:15:00Z"},
		{"id":13,"name":"unavailable log","status":"completed","conclusion":"success","completed_at":"2026-10-23T12:14:00Z","steps":[{"name":"Explain selection","conclusion":"success"}]}
	]}]`
	fake.responses[testBase+"/actions/jobs/10/logs"] = "2026-10-23T12:00:01Z " + decisionMarker + selectionSkip
	fake.responses[testBase+"/actions/jobs/11/logs"] = "2026-10-23T12:00:01Z " + decisionMarker + executionBlock
	var output, diagnostic bytes.Buffer
	require.NoError(t, refresh(t.Context(), fake.request, "stackrox/stackrox", 23602, 1, false, &output, &diagnostic))
	assert.Empty(t, fake.writes)
	assert.Contains(t, output.String(), "15m0s")
	assert.Contains(t, output.String(), "**Skipped by selection (1)**")
	assert.Contains(t, output.String(), "**Blocked by execution (1)**")
	assert.Contains(t, output.String(), "**No recorded decision (2)**")
	assert.Contains(t, output.String(), "decision log unavailable")
}

func TestGHRequestLogArguments(t *testing.T) {
	dir := t.TempDir()
	arguments := filepath.Join(dir, "arguments")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "gh"), []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$ARGUMENTS_FILE\"\nprintf 'response'\n"), 0o700))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("ARGUMENTS_FILE", arguments)
	for name, path := range map[string]string{
		"should read JSON through paginated slurp": testPR,
		"should read bounded raw log text":         testBase + "/actions/jobs/10/logs",
	} {
		t.Run(name, func(t *testing.T) {
			data, err := ghRequest(t.Context(), "GET", path, nil)
			require.NoError(t, err)
			assert.Equal(t, "response", string(data))
			args, err := os.ReadFile(arguments)
			require.NoError(t, err)
			if strings.HasSuffix(path, "/logs") {
				assert.Contains(t, string(args), "--allow-escape-sequences\n")
				assert.Contains(t, string(args), "Range: bytes=0-1048575\n")
				assert.NotContains(t, string(args), "--slurp")
			} else {
				assert.Contains(t, string(args), "--paginate\n--slurp\n")
			}
		})
	}
}

func TestRefreshUnchanged(t *testing.T) {
	fake := newFakeGitHub()
	var output, diagnostic bytes.Buffer
	require.NoError(t, refresh(t.Context(), fake.request, "stackrox/stackrox", 23602, 1, true, &output, &diagnostic))
	comments := []issueComment{{ID: 42, Body: fake.bodies[0]}}
	comments[0].User.Login = "github-actions[bot]"
	encoded, err := json.Marshal([][]issueComment{comments})
	require.NoError(t, err)
	fake.responses[testBase+"/issues/23602/comments?per_page=100"] = string(encoded)
	require.NoError(t, refresh(t.Context(), fake.request, "stackrox/stackrox", 23602, 1, true, &output, &diagnostic))
	assert.Len(t, fake.writes, 1)
}

func TestRefreshFailures(t *testing.T) {
	for name, setup := range map[string]func(*fakeGitHub){
		"should not post partial data when a job request fails": func(fake *fakeGitHub) {
			delete(fake.responses, testRun+"/jobs?per_page=100")
		},
		"should not treat malformed job JSON as zero duration": func(fake *fakeGitHub) {
			fake.responses[testRun+"/jobs?per_page=100"] = `{}`
		},
		"should reject a different PR": func(fake *fakeGitHub) {
			fake.responses[testRun] = `[{"event":"pull_request","run_attempt":1,"pull_requests":[{"number":999}]}]`
		},
		"should refuse ambiguous bot comments": func(fake *fakeGitHub) {
			fake.responses[testBase+"/issues/23602/comments?per_page=100"] = `[[
				{"id":1,"body":"` + commentMarker + `\nx","user":{"login":"github-actions[bot]"}},
				{"id":2,"body":"` + commentMarker + `\ny","user":{"login":"github-actions[bot]"}}
			]]`
		},
	} {
		t.Run(name, func(t *testing.T) {
			fake := newFakeGitHub()
			setup(fake)
			require.Error(t, refresh(t.Context(), fake.request, "stackrox/stackrox", 23602, 1, true, &bytes.Buffer{}, &bytes.Buffer{}))
			assert.Empty(t, fake.writes)
		})
	}
}

func TestCancelledWorkflow(t *testing.T) {
	fake := newFakeGitHub()
	fake.responses[testRun] = `[{"event":"pull_request","run_attempt":1,"conclusion":"cancelled","pull_requests":[{"number":23602}]}]`
	delete(fake.responses, testRun+"/jobs?per_page=100")
	require.NoError(t, refresh(t.Context(), fake.request, "stackrox/stackrox", 23602, 1, true, &bytes.Buffer{}, &bytes.Buffer{}))
	assert.Empty(t, fake.writes)
}

func TestRunArguments(t *testing.T) {
	for name, args := range map[string][]string{
		"should require a PR number":         {"--run=1"},
		"should require a workflow run ID":   {"--pr=1"},
		"should reject negative run IDs":     {"--pr=1", "--run=-1"},
		"should reject negative PR numbers":  {"--pr=-1", "--run=1"},
		"should reject repository paths":     {"--pr=1", "--run=1", "--repo=../x/y"},
		"should reject positional arguments": {"--pr=1", "--run=1", "extra"},
	} {
		t.Run(name, func(t *testing.T) {
			called := false
			request := func(context.Context, string, string, []byte) ([]byte, error) {
				called = true
				return nil, errors.New("unexpected request")
			}
			require.Error(t, run(t.Context(), args, request, &bytes.Buffer{}, &bytes.Buffer{}))
			assert.False(t, called)
		})
	}
}

func TestCancellation(t *testing.T) {
	fake := newFakeGitHub()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := run(ctx, []string{"--pr=23602", "--run=1"}, fake.request, &bytes.Buffer{}, &bytes.Buffer{})
	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, fake.writes)
}

func TestPublishingRequiresActions(t *testing.T) {
	for name, test := range map[string]struct{ actions, token string }{
		"should reject local publishing even with a token":  {actions: "false", token: "test-token"},
		"should reject publishing without a workflow token": {actions: "true"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("GITHUB_ACTIONS", test.actions)
			t.Setenv("GITHUB_TOKEN", test.token)
			fake := newFakeGitHub()
			err := run(t.Context(), []string{"--pr=23602", "--run=1", "--post"}, fake.request, &bytes.Buffer{}, &bytes.Buffer{})
			require.ErrorContains(t, err, "local execution is preview-only")
			assert.Empty(t, fake.writes)
		})
	}
	t.Run("should publish from Actions with its workflow token", func(t *testing.T) {
		t.Setenv("GITHUB_ACTIONS", "true")
		t.Setenv("GITHUB_TOKEN", "test-token")
		fake := newFakeGitHub()
		require.NoError(t, run(t.Context(), []string{"--pr=23602", "--run=1", "--post"}, fake.request, &bytes.Buffer{}, &bytes.Buffer{}))
		assert.Len(t, fake.writes, 1)
	})
}

func TestBelongsToPR(t *testing.T) {
	var head pullRequest
	head.Head.Ref = "piotr/prototype"
	head.Head.Repo.FullName = "stackrox/stackrox"
	for name, test := range map[string]struct {
		branch, repo string
		want         bool
	}{
		"should accept omitted association for the same source": {branch: "piotr/prototype", repo: "stackrox/stackrox", want: true},
		"should reject another branch":                          {branch: "another", repo: "stackrox/stackrox"},
		"should reject a fork with the same branch name":        {branch: "piotr/prototype", repo: "someone/stackrox"},
	} {
		t.Run(name, func(t *testing.T) {
			run := workflowRun{HeadBranch: test.branch}
			run.HeadRepo.FullName = test.repo
			assert.Equal(t, test.want, belongsToPR(run, 23602, head))
		})
	}
}
