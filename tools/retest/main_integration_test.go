package main

import (
	"context"
	_ "embed"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-github/v60/github"
	"github.com/stretchr/testify/assert"
)

func TestIntegration(t *testing.T) {
	var server *httptest.Server
	handler := func(w http.ResponseWriter, r *http.Request) {
		t.Logf("%s %s", r.Method, r.RequestURI)
		if r.Method == http.MethodPost {
			switch r.RequestURI {
			case "/repos/stackrox/stackrox/issues/132/comments":
				b, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				assert.JSONEq(t, `{"body":"/retest"}`, string(b))
				_, err = w.Write([]byte(`{"html_url": "some url"}`))
				assert.NoError(t, err)
			case "/repos/stackrox/stackrox/issues/2/comments":
				b, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				assert.JSONEq(t, `{"body":"/test job-name-1\n/test job-name-2"}`, string(b))
				_, err = w.Write([]byte(`{"html_url": "some url"}`))
				assert.NoError(t, err)
			case "/repos/stackrox/stackrox/issues/500/comments":
				b, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				assert.JSONEq(t,
					`{"body":":x: There was an error with a comment. Please edit or remove it and issue a proper command\ngot an error in a comment \"/retest-times 10000000000000000000000000000 job-name-1\": strconv.Atoi: parsing \"10000000000000000000000000000\": value out of range"}`,
					string(b))
				_, err = w.Write([]byte(`{"html_url": "some url"}`))
				assert.NoError(t, err)
			default:
				assert.Failf(t, "unexpected call ", r.RequestURI)
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
			assert.NoError(t, err)
		case "/user":
			_, err := w.Write([]byte(`{
            "login": "octocat",
            "html_url": "https://github.com/octocat"
        }`))
			assert.NoError(t, err)
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
			assert.NoError(t, err)
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
			assert.NoError(t, err)
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
			assert.NoError(t, err)
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
			assert.NoError(t, err)
		case "/repos/stackrox/stackrox/pulls/132", "/repos/stackrox/stackrox/pulls/2", "/repos/stackrox/stackrox/pulls/500", "/repos/stackrox/stackrox/pulls/501":
			_, err := w.Write([]byte(`{
    "html_url": "https://github.com/octocat/Hello-World/pull/1347",
    "number": 132,
    "head": {
        "sha": "6dcb09b5b57875f334f61aebed695e2e4193db5e"
    },
	"statuses_url": "` + server.URL + `/repos/octocat/Hello-World/statuses/6dcb09b5b57875f334f61aebed695e2e4193db5e"
}`))
			assert.NoError(t, err)
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
			assert.NoError(t, err)
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
			assert.NoError(t, err)
		case `/repos/octocat/Hello-World/statuses/6dcb09b5b57875f334f61aebed695e2e4193db5e`:
			_, err := w.Write([]byte(`[
  {
    "state": "failure",
    "context": "ci/prow/gke-upgrade-tests"
}]`))
			assert.NoError(t, err)
		default:
			assert.Failf(t, "unexpected call ", r.RequestURI)
			w.WriteHeader(http.StatusNotFound)
		}
	}
	server = httptest.NewServer(http.HandlerFunc(handler))
	t.Cleanup(server.Close)

	client := github.NewClient(server.Client())
	parse, err := url.Parse(server.URL + "/")
	assert.NoError(t, err)
	client.BaseURL = parse

	err = run(context.Background(), client)
	assert.NoError(t, err)

}

//go:embed testdata/statuses.json
var statusesResponse []byte

func TestGetStatuses(t *testing.T) {
	var server *httptest.Server
	handler := func(w http.ResponseWriter, r *http.Request) {
		t.Logf("%s %s", r.Method, r.RequestURI)
		_, err := w.Write(statusesResponse)
		assert.NoError(t, err)
	}
	server = httptest.NewServer(http.HandlerFunc(handler))
	t.Cleanup(server.Close)

	client := github.NewClient(server.Client())
	baseUrl, err := url.Parse(server.URL + "/")
	assert.NoError(t, err)
	client.BaseURL = baseUrl

	statuses, err := statusesForPR(context.Background(), client, baseUrl.String())
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"gke-upgrade-tests": "success"}, statuses)
}
