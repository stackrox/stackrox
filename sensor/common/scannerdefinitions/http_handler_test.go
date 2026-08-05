package scannerdefinitions

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/scannerv4/repositorytocpe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServeHTTP_Responses(t *testing.T) {
	type args struct {
		writer  *httptest.ResponseRecorder
		request *http.Request
		methods []string
	}
	tests := []struct {
		name             string
		args             args
		responseBody     string
		jsonResponse     bool
		statusCode       int
		centralReachable bool
	}{
		{
			name:             "when central is not reachable then return internal error",
			statusCode:       http.StatusServiceUnavailable,
			responseBody:     "{\"code\":14,\"message\":\"central not reachable\"}",
			jsonResponse:     true,
			centralReachable: false,
		},
		{
			name:             "when central replies 200 with content then writer matches",
			statusCode:       http.StatusOK,
			responseBody:     "the foobar body.",
			centralReachable: true,
		},
		{
			name:             "when central replies 304 then writer matches",
			statusCode:       http.StatusNotModified,
			centralReachable: true,
		},
		{
			name:       "when method is not GET then 405",
			statusCode: http.StatusMethodNotAllowed,
			args: args{
				methods: []string{
					http.MethodHead,
					http.MethodPost,
					http.MethodPut,
					http.MethodPatch,
					http.MethodDelete,
					http.MethodConnect,
					http.MethodOptions,
					http.MethodTrace,
				},
				request: &http.Request{},
			},
			centralReachable: true,
		},
		{
			name:       "when request contains multiple headers then proxy all of them",
			statusCode: http.StatusOK,
			args: args{
				request: &http.Request{
					URL:    &url.URL{},
					Header: map[string][]string{"Accept-Encoding": {"foo", "bar"}},
				},
			},
			centralReachable: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set args defaults.
			if tt.args.writer == nil {
				tt.args.writer = httptest.NewRecorder()
			}
			if tt.args.methods == nil {
				// Defaults to GET.
				tt.args.methods = []string{http.MethodGet}
			}
			// Perform one test per HTTP method.
			for _, method := range tt.args.methods {
				if tt.args.request == nil {
					tt.args.request = &http.Request{
						Method: method,
						URL:    &url.URL{RawQuery: "bar=1&foo=2"},
						Header: map[string][]string{"If-Modified-Since": {"1209"}, "Accept-Encoding": {""}},
					}
				} else {
					tt.args.request.Method = method
				}
				h := &Handler{
					centralClient: &http.Client{
						Transport: httputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
							assert.Equal(t, tt.args.request.URL.RawQuery, req.URL.RawQuery)
							for header := range headersToProxy.All() {
								assert.Equal(t, tt.args.request.Header.Values(header), req.Header.Values(header))
							}
							return &http.Response{
								StatusCode: tt.statusCode,
								Body:       io.NopCloser(bytes.NewBufferString(tt.responseBody)),
							}, nil
						}),
					},
				}
				h.centralReachable.Store(tt.centralReachable)
				h.ServeHTTP(tt.args.writer, tt.args.request)
				if tt.jsonResponse {
					assert.JSONEq(t, tt.responseBody, tt.args.writer.Body.String())
				} else {
					assert.Equal(t, tt.responseBody, tt.args.writer.Body.String())
				}
				assert.Equal(t, tt.statusCode, tt.args.writer.Code)
			}
		})
	}
}

// newTestCentralClient spins up a real httptest.Server and returns a client
// whose transport rewrites relative URLs to point at it, mirroring how the
// production Central transport fills in scheme/host for requests built with
// only a path (see newRepo2CPERequest).
func newTestCentralClient(t *testing.T, handler http.HandlerFunc) *http.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	base, err := url.Parse(server.URL)
	require.NoError(t, err)

	client := server.Client()
	client.Transport = httputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req = req.Clone(req.Context())
		req.URL.Scheme = base.Scheme
		req.URL.Host = base.Host
		return http.DefaultTransport.RoundTrip(req)
	})
	return client
}

func TestAttemptRepo2CPERefresh(t *testing.T) {
	const mappingV1 = `{"data":{"foo":{"cpes":["cpe:/o:foo"]}}}`
	const mappingV2 = `{"data":{"bar":{"cpes":["cpe:/o:bar"]}}}`
	hashV1 := repositorytocpe.HashMapping([]byte(mappingV1))
	hashV2 := repositorytocpe.HashMapping([]byte(mappingV2))

	tests := map[string]struct {
		seedCache     repo2CPECache
		serverHandler http.HandlerFunc
		networkErr    bool
		wantOK        bool
		wantMapping   string
		wantHash      string
	}{
		"first fetch succeeds and populates an empty cache": {
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, repo2CPEFileParam, r.URL.Query().Get("file"))
				w.Header().Set(etagHeader, `"v1"`)
				_, _ = w.Write([]byte(mappingV1))
			},
			wantOK:      true,
			wantMapping: mappingV1,
			wantHash:    hashV1,
		},
		"304 with matching validators keeps the cached mapping": {
			seedCache: repo2CPECache{mapping: []byte(mappingV1), hash: hashV1, etag: `"v1"`},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, `"v1"`, r.Header.Get(ifNoneMatchHeader))
				w.WriteHeader(http.StatusNotModified)
			},
			wantOK:      true,
			wantMapping: mappingV1,
			wantHash:    hashV1,
		},
		"200 with a same-hash body is treated as unchanged": {
			seedCache: repo2CPECache{mapping: []byte(mappingV1), hash: hashV1},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(mappingV1))
			},
			wantOK:      true,
			wantMapping: mappingV1,
			wantHash:    hashV1,
		},
		"200 with a different hash replaces the cached mapping": {
			seedCache: repo2CPECache{mapping: []byte(mappingV1), hash: hashV1},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(mappingV2))
			},
			wantOK:      true,
			wantMapping: mappingV2,
			wantHash:    hashV2,
		},
		"an unexpected status leaves the cache untouched": {
			seedCache: repo2CPECache{mapping: []byte(mappingV1), hash: hashV1},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantOK:      false,
			wantMapping: mappingV1,
			wantHash:    hashV1,
		},
		"a network error leaves the cache untouched": {
			seedCache:   repo2CPECache{mapping: []byte(mappingV1), hash: hashV1},
			networkErr:  true,
			wantOK:      false,
			wantMapping: mappingV1,
			wantHash:    hashV1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			h := &Handler{cache: tt.seedCache}
			if tt.networkErr {
				h.centralClient = &http.Client{
					Transport: httputil.RoundTripperFunc(func(_ *http.Request) (*http.Response, error) {
						return nil, errors.New("connection refused")
					}),
				}
			} else {
				h.centralClient = newTestCentralClient(t, tt.serverHandler)
			}

			ok := h.attemptRepo2CPERefresh(context.Background())

			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantMapping, string(h.cache.mapping))
			assert.Equal(t, tt.wantHash, h.cache.hash)
			assert.False(t, h.cache.lastAttempt.IsZero(), "lastAttempt should be recorded regardless of outcome")
			if tt.wantOK {
				assert.False(t, h.cache.lastSuccess.IsZero())
			} else {
				assert.True(t, h.cache.lastSuccess.IsZero())
			}
		})
	}
}

// TestAttemptRepo2CPERefresh_RetryRecoversWithoutLosingCache calls the
// refresh helper directly (no timers/sleeps) to verify a failed attempt
// keeps serving the last good mapping and a later success can recover.
func TestAttemptRepo2CPERefresh_RetryRecoversWithoutLosingCache(t *testing.T) {
	const mapping = `{"data":{"foo":{"cpes":["cpe:/o:foo"]}}}`
	hash := repositorytocpe.HashMapping([]byte(mapping))

	var fail atomic.Bool
	h := &Handler{centralClient: newTestCentralClient(t, func(w http.ResponseWriter, _ *http.Request) {
		if fail.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(mapping))
	})}

	require.True(t, h.attemptRepo2CPERefresh(context.Background()))
	assert.Equal(t, hash, h.cache.hash)

	fail.Store(true)
	require.False(t, h.attemptRepo2CPERefresh(context.Background()))
	assert.Equal(t, hash, h.cache.hash, "a failed refresh must not drop the last good mapping")

	fail.Store(false)
	require.True(t, h.attemptRepo2CPERefresh(context.Background()))
	assert.Equal(t, hash, h.cache.hash)
}

func TestFetchRepo2CPE(t *testing.T) {
	mapping := []byte(`{"data":{"foo":{"cpes":["cpe:/o:foo"]}}}`)
	hash := repositorytocpe.HashMapping(mapping)

	t.Run("never fetched returns ok=false", func(t *testing.T) {
		h := &Handler{centralClient: &http.Client{}}
		h.centralReachable.Store(false)

		gotMapping, gotHash, ok := h.FetchRepo2CPE(context.Background())

		assert.False(t, ok)
		assert.Nil(t, gotMapping)
		assert.Empty(t, gotHash)
	})

	t.Run("offline serves the last cached mapping with ok=true", func(t *testing.T) {
		h := &Handler{centralClient: &http.Client{}}
		h.cache = repo2CPECache{mapping: mapping, hash: hash, lastSuccess: time.Now()}
		h.centralReachable.Store(false)

		gotMapping, gotHash, ok := h.FetchRepo2CPE(context.Background())

		assert.True(t, ok)
		assert.Equal(t, mapping, gotMapping)
		assert.Equal(t, hash, gotHash)
	})
}
