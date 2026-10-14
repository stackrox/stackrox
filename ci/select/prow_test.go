package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommentRequests(t *testing.T) {
	for body, want := range map[string]bool{
		"/test unit": true, " /test all ": true, "hi\n/test unit e2e": true,
		"/test unity": false, "/test": false, "quoted /test unit": false,
		"/testing unit": false, "/test,unit": false,
	} {
		t.Run(body, func(t *testing.T) { assert.Equal(t, want, commentRequests("unit", body)) })
	}
}

func TestRequestsSince(t *testing.T) {
	for name, test := range map[string]struct {
		created, commit string
		want            bool
	}{
		"equal":           {created: "2026-09-30T12:00:00Z", commit: "2026-09-30T12:00:00Z", want: true},
		"older":           {created: "2026-09-30T11:59:59Z", commit: "2026-09-30T12:00:00Z"},
		"offset":          {created: "2026-09-30T14:00:00+02:00", commit: "2026-09-30T12:00:00Z", want: true},
		"invalid comment": {created: "bad", commit: "2026-09-30T12:00:00Z"},
		"invalid commit":  {created: "2026-09-30T12:00:00Z", commit: "bad"},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.want, requestsSince([]comment{{CreatedAt: test.created, Body: "/test unit"}}, "unit", test.commit))
		})
	}
}

func TestProwCI(t *testing.T) {
	t.Setenv("CI_SELECT_ROOT", "../..")
	t.Setenv("PULL_BASE_SHA", "missing-ref")
	t.Setenv("PULL_PULL_SHA", "HEAD")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "[]")
	}))
	defer server.Close()
	t.Setenv("GITHUB_API_URL", server.URL)
	input := `{"number":23166,"labels":[{"name":"ci-dispatcher-enforce"}],"base":{"repo":{"full_name":"stackrox/stackrox"}}}`
	code, err := run(t.Context(), []string{"prow-ci", "gke-qa-e2e-tests"}, strings.NewReader(input), &bytes.Buffer{}, &bytes.Buffer{})
	require.NoError(t, err)
	assert.Equal(t, 3, code)
	code, err = run(t.Context(), []string{"prow-ci", "unknown-job"}, strings.NewReader(input), &bytes.Buffer{}, &bytes.Buffer{})
	require.NoError(t, err)
	assert.Zero(t, code)
	code, err = run(t.Context(), []string{"prow-ci", "unit"}, strings.NewReader("invalid"), &bytes.Buffer{}, &bytes.Buffer{})
	require.Error(t, err)
	assert.NotEqual(t, 3, code)
}

func TestProwCIComments(t *testing.T) {
	t.Setenv("CI_SELECT_ROOT", "../..")
	t.Setenv("PULL_BASE_SHA", "missing-ref")
	t.Setenv("PULL_PULL_SHA", "HEAD")
	input := `{"number":23166,"labels":[{"name":"ci-dispatcher-enforce"}],"base":{"repo":{"full_name":"stackrox/stackrox"}}}`
	for name, test := range map[string]struct {
		status    int
		body      string
		wantError bool
	}{
		"test forces run":    {status: http.StatusOK, body: `[{"created_at":"2099-01-01T00:00:00Z","body":"/test gke-qa-e2e-tests"}]`},
		"API unavailable":    {status: http.StatusForbidden, body: `{}`, wantError: true},
		"malformed response": {status: http.StatusOK, body: `{}`, wantError: true},
		"null response":      {status: http.StatusOK, body: `null`, wantError: true},
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.status)
				fmt.Fprint(w, test.body)
			}))
			defer server.Close()
			t.Setenv("GITHUB_API_URL", server.URL)
			code, err := run(t.Context(), []string{"prow-ci", "gke-qa-e2e-tests"}, strings.NewReader(input),
				&bytes.Buffer{}, &bytes.Buffer{})
			assert.NotEqual(t, 3, code)
			if test.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Zero(t, code)
			}
		})
	}
}

func TestCommentsPagination(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		comments := []comment{}
		if r.URL.Query().Get("page") == "1" {
			comments = make([]comment, 100)
		} else {
			comments = append(comments, comment{CreatedAt: "2026-09-30T12:00:00Z", Body: "/test unit"})
		}
		assert.NoError(t, json.NewEncoder(w).Encode(comments))
	}))
	defer server.Close()
	g := githubClient{baseURL: server.URL, client: server.Client()}
	comments, err := g.comments(t.Context(), "stackrox/stackrox", "23166")
	require.NoError(t, err)
	assert.Equal(t, 2, calls)
	assert.True(t, requestsSince(comments, "unit", "2026-09-30T11:00:00Z"))
}

func TestProwWrapper(t *testing.T) {
	for name, status := range map[string]int{"run": 0, "skip": 3, "failure": 1} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeTestFile(t, root+"/ci/select/bin/ci-select", fmt.Sprintf("#!/bin/sh\nexit %d\n", status), 0o700)
			command := exec.CommandContext(t.Context(), "bash", "-c",
				`source ./prow-gate.sh; get_pr_details() { echo '{}'; }; ci_decision_allows unit`)
			command.Env = append(command.Environ(), "ROOT="+root)
			err := command.Run()
			assert.Equal(t, status == 3, err != nil)
		})
	}
}
