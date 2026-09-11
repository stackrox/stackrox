package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v91/github"
)

func TestIntegration(t *testing.T) {
	var server *httptest.Server
	handler := func(w http.ResponseWriter, r *http.Request) {
		t.Logf("%s %s", r.Method, r.RequestURI)
		if r.Method == http.MethodPost {
			switch r.RequestURI {
			case "/repos/stackrox/stackrox/issues/132/comments":
				b, err := io.ReadAll(r.Body)
				assertNoError(t, err)
				assertJSONEq(t, `{"body":"/retest"}`, string(b))
				_, err = w.Write([]byte(`{"html_url": "some url"}`))
				assertNoError(t, err)
			case "/repos/stackrox/stackrox/issues/2/comments":
				b, err := io.ReadAll(r.Body)
				assertNoError(t, err)
				assertJSONEq(t, `{"body":"/test job-name-1\n/test job-name-2"}`, string(b))
				_, err = w.Write([]byte(`{"html_url": "some url"}`))
				assertNoError(t, err)
			case "/repos/stackrox/stackrox/issues/500/comments":
				b, err := io.ReadAll(r.Body)
				assertNoError(t, err)
				assertJSONEq(t,
					`{"body":":x: There was an error with a comment. Please edit or remove it and issue a proper command\ngot an error in a comment \"/retest-times 10000000000000000000000000000 job-name-1\": strconv.Atoi: parsing \"10000000000000000000000000000\": value out of range"}`,
					string(b))
				_, err = w.Write([]byte(`{"html_url": "some url"}`))
				assertNoError(t, err)
			default:
				t.Errorf("unexpected call %s", r.RequestURI)
			}
			return
		}

		switch r.RequestURI {
		case `/search/issues?q=repo%3Astackrox%2Fstackrox+label%3Aauto-retest+state%3Aopen+type%3Apr`:
			_, err := w.Write([]byte(`
{
  "total_count": 2,
  "incomplete_results": false,
  "items": [
    {
      "comments_url": "https://api.github.com/repos/batterseapower/pinyin-toolkit/issues/132/comments",
      "html_url": "https://github.com/batterseapower/pinyin-toolkit/issues/132",
      "number": 132
    },
    {
      "comments_url": "https://api.github.com/repos/batterseapower/pinyin-toolkit/issues/132/comments",
      "html_url": "https://github.com/batterseapower/pinyin-toolkit/issues/132",
      "number": 2
    },
    {
      "comments_url": "https://api.github.com/repos/batterseapower/pinyin-toolkit/issues/132/comments",
      "html_url": "https://github.com/batterseapower/pinyin-toolkit/issues/132",
      "number": 404
    },
    {
      "comments_url": "https://api.github.com/repos/batterseapower/pinyin-toolkit/issues/132/comments",
      "html_url": "https://github.com/batterseapower/pinyin-toolkit/issues/132",
      "number": 500
    },
    {
      "comments_url": "https://api.github.com/repos/batterseapower/pinyin-toolkit/issues/132/comments",
      "html_url": "https://github.com/batterseapower/pinyin-toolkit/issues/132",
      "number": 501
    }
  ]
}`))
			assertNoError(t, err)
		case "/user":
			_, err := w.Write([]byte(`{
            "login": "octocat",
            "html_url": "https://github.com/octocat"
        }`))
			assertNoError(t, err)
		case "/repos/stackrox/stackrox/issues/2/comments?direction=asc&sort=created":
			_, err := w.Write([]byte(`[
    {
        "id": 1,
        "html_url": "https://github.com/octocat/Hello-World/issues/1347#issuecomment-1",
        "body": "/retest-times 10 job-name-1\n/retest-times 20 job-name-2\n",
        "user": {
            "login": "octocat",
            "html_url": "https://github.com/octocat"
        }
    }
]`))
			assertNoError(t, err)
		case "/repos/stackrox/stackrox/issues/500/comments?direction=asc&sort=created":
			_, err := w.Write([]byte(`[
    {
        "id": 1,
        "html_url": "https://github.com/octocat/Hello-World/issues/1347#issuecomment-1",
        "body": "/retest-times 10000000000000000000000000000 job-name-1",
        "user": {
            "login": "octocat",
            "html_url": "https://github.com/octocat"
        }
    }
]`))
			assertNoError(t, err)
		case "/repos/stackrox/stackrox/issues/501/comments?direction=asc&page=2&sort=created":
			_, err := w.Write([]byte(`[
    {
        "id": 1,
        "html_url": "https://github.com/octocat/Hello-World/issues/1347#issuecomment-2",
        "body": ":x: There was an error with a comment. Please edit or remove it and issue a proper command\ngot an error in a comment \"/retest-times 10000000000000000000000000000 job-name-1\": strconv.Atoi: parsing \"10000000000000000000000000000\": value out of range",
        "user": {
            "login": "octocat",
            "html_url": "https://github.com/octocat"
        }
    }
]`))
			assertNoError(t, err)
		case "/repos/stackrox/stackrox/issues/501/comments?direction=asc&sort=created":
			w.Header().Set("link", `<?page=2>; rel="next";`)
			_, err := w.Write([]byte(`[
    {
        "id": 1,
        "html_url": "https://github.com/octocat/Hello-World/issues/1347#issuecomment-1",
        "body": "/retest-times 10000000000000000000000000000 job-name-1",
        "user": {
            "login": "octocat",
            "html_url": "https://github.com/octocat"
        }
    }
]`))
			assertNoError(t, err)
		case "/repos/stackrox/stackrox/pulls/132", "/repos/stackrox/stackrox/pulls/2", "/repos/stackrox/stackrox/pulls/500", "/repos/stackrox/stackrox/pulls/501":
			_, err := w.Write([]byte(`{
    "html_url": "https://github.com/octocat/Hello-World/pull/1347",
    "number": 132,
    "head": {
        "sha": "6dcb09b5b57875f334f61aebed695e2e4193db5e"
    },
	"statuses_url": "` + server.URL + `/repos/octocat/Hello-World/statuses/6dcb09b5b57875f334f61aebed695e2e4193db5e"
}`))
			assertNoError(t, err)
		case "/repos/stackrox/stackrox/issues/132/comments?direction=asc&sort=created":
			_, err := w.Write([]byte(`[
    {
        "id": 1,
        "html_url": "https://github.com/octocat/Hello-World/issues/1347#issuecomment-1",
        "body": "Me too",
        "user": {
            "login": "octocat",
            "html_url": "https://github.com/octocat"
        }
    }
]`))
			assertNoError(t, err)
		case "/repos/stackrox/stackrox/pulls/404":
			http.NotFound(w, r)
		case `/repos/stackrox/stackrox/commits/6dcb09b5b57875f334f61aebed695e2e4193db5e/check-runs?filter=latest&status=completed`:
			_, err := w.Write([]byte(`{
 "total_count": 2,
  "check_runs": [
    {
      "html_url": "https://github.com/github/hello-world/runs/4",
      "status": "completed",
      "conclusion": "neutral",
      "name": "CI"
    },
    {
      "html_url": "https://github.com/github/hello-world/runs/4",
      "status": "completed",
      "conclusion": "success",
      "name": "prow"
    }
  ]
}`))
			assertNoError(t, err)
		case `/repos/octocat/Hello-World/statuses/6dcb09b5b57875f334f61aebed695e2e4193db5e`:
			_, err := w.Write([]byte(`[
  {
    "state": "failure",
    "context": "ci/prow/gke-upgrade-tests"
}]`))
			assertNoError(t, err)
		default:
			t.Errorf("unexpected call %s", r.RequestURI)
			w.WriteHeader(http.StatusNotFound)
		}
	}
	server = httptest.NewServer(http.HandlerFunc(handler))
	t.Cleanup(server.Close)

	baseURL := server.URL + "/"
	client, err := github.NewClient(github.WithHTTPClient(server.Client()), github.WithURLs(&baseURL, nil))
	mustNoError(t, err)

	err = run(context.Background(), client)
	assertNoError(t, err)

}

//go:embed testdata/statuses.json
var statusesResponse []byte

func TestGetStatuses(t *testing.T) {
	var server *httptest.Server
	handler := func(w http.ResponseWriter, r *http.Request) {
		t.Logf("%s %s", r.Method, r.RequestURI)
		_, err := w.Write(statusesResponse)
		assertNoError(t, err)
	}
	server = httptest.NewServer(http.HandlerFunc(handler))
	t.Cleanup(server.Close)

	baseURL2 := server.URL + "/"
	client, err := github.NewClient(github.WithHTTPClient(server.Client()), github.WithURLs(&baseURL2, nil))
	mustNoError(t, err)

	statuses, err := statusesForPR(context.Background(), client, baseURL2)
	assertNoError(t, err)
	want := map[string]jobState{"gke-upgrade-tests": jobOK}
	if !maps.Equal(statuses, want) {
		t.Errorf("statuses = %v, want %v", statuses, want)
	}
}

func mustNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func assertJSONEq(t *testing.T, want, got string) {
	t.Helper()
	want, got = canonicalJSON(t, want), canonicalJSON(t, got)
	if want != got {
		t.Errorf("JSON mismatch:\n got: %s\nwant: %s", got, want)
	}
}

func canonicalJSON(t *testing.T, raw string) string {
	t.Helper()

	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		t.Fatalf("invalid JSON %q: %v", raw, err)
	}
	normalized, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("could not normalize JSON: %v", err)
	}
	return string(normalized)
}
