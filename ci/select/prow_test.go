package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
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

func TestProwCIRehearsalEnforcement(t *testing.T) {
	t.Setenv("CI_SELECT_ROOT", "../..")
	t.Setenv("PULL_BASE_SHA", "missing-ref")
	t.Setenv("PULL_PULL_SHA", "HEAD")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "[]")
	}))
	defer server.Close()
	t.Setenv("GITHUB_API_URL", server.URL)
	for body, want := range map[string]string{
		"/label: ci-dispatcher-enforce":                "skip",
		"description\n/label: ci-dispatcher-enforce\n": "skip",
		"  /label:\tci-dispatcher-enforce\r\n":         "skip",
		"/label: ci-dispatcher-enforce another-label":  "skip",
		"/label: ci-dispatcher-enforced":               "run",
		"/label: ci-dispatcher-enforce-extra":          "run",
		"/label: ci-dispatcher-enforce_extra":          "run",
		"/label: ci-dispatcher-enforce.extra":          "run",
		"/label: ci-dispatcher-enforce/extra":          "run",
		"quoted /label: ci-dispatcher-enforce":         "run",
		"/label:\nci-dispatcher-enforce":               "run",
		"":                                             "run",
	} {
		t.Run(body, func(t *testing.T) {
			pr := pullRequest{Number: 23403, Body: body}
			pr.Base.Repo.FullName = "openshift/release"
			input, err := json.Marshal(pr)
			require.NoError(t, err)
			action, err := prowCI(t.Context(), "gke-qa-e2e-tests", bytes.NewReader(input), &bytes.Buffer{})
			require.NoError(t, err)
			assert.Equal(t, want, action)
		})
	}
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

func TestProwDispatchUIOnly(t *testing.T) {
	dispatch, err := os.ReadFile("../../.openshift-ci/dispatch.sh")
	require.NoError(t, err)
	gate, err := os.ReadFile("prow-gate.sh")
	require.NoError(t, err)
	for name, test := range map[string]struct {
		job                          string
		pullRequest, enforce, uiOnly bool
		buildStatus, gateStatus      int
		wantDispatch, wantBuild      bool
	}{
		"should skip unenforced UI-only nongroovy": {
			job: "gke-nongroovy-e2e-tests", pullRequest: true, uiOnly: true,
		},
		"should skip unenforced UI-only upgrade": {
			job: "gke-upgrade-tests", pullRequest: true, uiOnly: true,
		},
		"should skip unenforced UI-only even when selector cannot build": {
			job: "gke-nongroovy-e2e-tests", pullRequest: true, uiOnly: true, buildStatus: 1,
		},
		"should run UI tests on unenforced UI-only changes": {
			job: "gke-ui-e2e-tests", pullRequest: true, uiOnly: true, wantDispatch: true, wantBuild: true,
		},
		"should run unenforced mixed changes": {
			job: "gke-nongroovy-e2e-tests", pullRequest: true, wantDispatch: true, wantBuild: true,
		},
		"should follow enforced run on UI-only changes": {
			job: "gke-nongroovy-e2e-tests", pullRequest: true, enforce: true, uiOnly: true,
			wantDispatch: true, wantBuild: true,
		},
		"should follow enforced skip": {
			job: "gke-nongroovy-e2e-tests", pullRequest: true, enforce: true, uiOnly: true,
			gateStatus: 3, wantBuild: true,
		},
		"should run when enforced selector build fails": {
			job: "gke-nongroovy-e2e-tests", pullRequest: true, enforce: true, uiOnly: true,
			buildStatus: 1, wantDispatch: true, wantBuild: true,
		},
		"should run outside PR context": {
			job: "gke-nongroovy-e2e-tests", uiOnly: true, wantDispatch: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeTestFile(t, filepath.Join(root, ".openshift-ci/dispatch.sh"), string(dispatch), 0o700)
			writeTestFile(t, filepath.Join(root, "scripts/ci/lib.sh"), `is_in_PR_context() { [[ "$CI_SELECT_TEST_PR" == true ]]; }
pr_has_label() { [[ "$CI_SELECT_TEST_ENFORCE" == true ]]; }
changes_limited_to() { [[ "$CI_SELECT_TEST_UI_ONLY" == true ]]; }
info() { echo "$*"; }
get_pr_details() { echo '{}'; }
openshift_ci_mods() { echo 'job dispatched'; exit 0; }
`, 0o600)
			writeTestFile(t, filepath.Join(root, "tests/e2e/lib.sh"), "", 0o600)
			writeTestFile(t, filepath.Join(root, "ci/select/build.sh"), "#!/bin/sh\necho 'selector build attempted'\nexit \"$CI_SELECT_TEST_BUILD_STATUS\"\n", 0o700)
			writeTestFile(t, filepath.Join(root, "ci/select/prow-gate.sh"), string(gate), 0o600)
			writeTestFile(t, filepath.Join(root, "ci/select/bin/ci-select"), "#!/bin/sh\nexit \"$CI_SELECT_TEST_GATE_STATUS\"\n", 0o700)
			command := exec.CommandContext(t.Context(), "bash", filepath.Join(root, ".openshift-ci/dispatch.sh"), test.job)
			command.Env = append(command.Environ(),
				"ARTIFACT_DIR=",
				fmt.Sprintf("CI_SELECT_TEST_PR=%t", test.pullRequest),
				fmt.Sprintf("CI_SELECT_TEST_ENFORCE=%t", test.enforce),
				fmt.Sprintf("CI_SELECT_TEST_UI_ONLY=%t", test.uiOnly),
				fmt.Sprintf("CI_SELECT_TEST_BUILD_STATUS=%d", test.buildStatus),
				fmt.Sprintf("CI_SELECT_TEST_GATE_STATUS=%d", test.gateStatus),
			)
			output, err := command.CombinedOutput()
			require.NoError(t, err, string(output))
			assert.Equal(t, test.wantDispatch, strings.Contains(string(output), "job dispatched"), string(output))
			assert.Equal(t, test.wantBuild, strings.Contains(string(output), "selector build attempted"), string(output))
		})
	}
}
