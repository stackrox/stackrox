package scannerdefinitions

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/virtualmachine/cpemapping"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testRepo2CPEMapping = `{"data":{"foo":{"cpes":["cpe:/o:foo"]}}}`

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

func countingMappingClient(fetches *atomic.Int32, mapping string) *http.Client {
	return &http.Client{
		Transport: httputil.RoundTripperFunc(func(_ *http.Request) (*http.Response, error) {
			fetches.Add(1)
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(bytes.NewBufferString(mapping)),
			}, nil
		}),
	}
}

func TestAttemptRepo2CPERefresh(t *testing.T) {
	const mappingV1 = `{"data":{"foo":{"cpes":["cpe:/o:foo"]}}}`
	const mappingV2 = `{"data":{"bar":{"cpes":["cpe:/o:bar"]}}}`
	hashV1 := cpemapping.HashMapping([]byte(mappingV1))
	hashV2 := cpemapping.HashMapping([]byte(mappingV2))
	seededSuccess := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := map[string]struct {
		seedCache        repo2CPECache
		serverHandler    http.HandlerFunc
		networkErr       bool
		wantOK           bool
		wantMapping      string
		wantHash         string
		wantEtag         string
		wantLastModified string
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
			wantEtag:    `"v1"`,
		},
		"304 with matching validators keeps the cached mapping": {
			seedCache: repo2CPECache{mapping: []byte(mappingV1), hash: hashV1, etag: `"v1"`, lastModified: "old", lastSuccess: seededSuccess},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, `"v1"`, r.Header.Get(ifNoneMatchHeader))
				w.WriteHeader(http.StatusNotModified)
			},
			wantOK:           true,
			wantMapping:      mappingV1,
			wantHash:         hashV1,
			wantEtag:         `"v1"`,
			wantLastModified: "old",
		},
		"200 without validators replaces previously cached validators": {
			seedCache: repo2CPECache{mapping: []byte(mappingV1), hash: hashV1, etag: `"v1"`, lastModified: "old", lastSuccess: seededSuccess},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(mappingV2))
			},
			wantOK:      true,
			wantMapping: mappingV2,
			wantHash:    hashV2,
		},
		"200 with a same-hash body is treated as unchanged": {
			seedCache: repo2CPECache{mapping: []byte(mappingV1), hash: hashV1, lastSuccess: seededSuccess},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(mappingV1))
			},
			wantOK:      true,
			wantMapping: mappingV1,
			wantHash:    hashV1,
		},
		"200 with a different hash replaces the cached mapping": {
			seedCache: repo2CPECache{mapping: []byte(mappingV1), hash: hashV1, lastSuccess: seededSuccess},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(mappingV2))
			},
			wantOK:      true,
			wantMapping: mappingV2,
			wantHash:    hashV2,
		},
		"an unexpected status leaves the cache untouched": {
			seedCache: repo2CPECache{mapping: []byte(mappingV1), hash: hashV1, lastSuccess: seededSuccess},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantOK:      false,
			wantMapping: mappingV1,
			wantHash:    hashV1,
		},
		"a network error leaves the cache untouched": {
			seedCache:   repo2CPECache{mapping: []byte(mappingV1), hash: hashV1, lastSuccess: seededSuccess},
			networkErr:  true,
			wantOK:      false,
			wantMapping: mappingV1,
			wantHash:    hashV1,
		},
		"an oversized response leaves the cache untouched": {
			seedCache: repo2CPECache{mapping: []byte(mappingV1), hash: hashV1, lastSuccess: seededSuccess},
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write(bytes.Repeat([]byte("a"), cpemapping.MaxMappingBytes+1))
			},
			wantOK:      false,
			wantMapping: mappingV1,
			wantHash:    hashV1,
		},
		"an invalid mapping body leaves the cache untouched": {
			seedCache: repo2CPECache{mapping: []byte(mappingV1), hash: hashV1, lastSuccess: seededSuccess},
			serverHandler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"not":"a mapping"}`))
			},
			wantOK:      false,
			wantMapping: mappingV1,
			wantHash:    hashV1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r := &Repo2CPE{cache: tt.seedCache}
			if tt.networkErr {
				r.centralClient = &http.Client{
					Transport: httputil.RoundTripperFunc(func(_ *http.Request) (*http.Response, error) {
						return nil, errors.New("connection refused")
					}),
				}
			} else {
				r.centralClient = newTestCentralClient(t, tt.serverHandler)
			}

			ok := r.attemptRepo2CPERefresh(t.Context())

			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantMapping, string(r.cache.mapping))
			assert.Equal(t, tt.wantHash, r.cache.hash)
			assert.Equal(t, tt.wantEtag, r.cache.etag)
			assert.Equal(t, tt.wantLastModified, r.cache.lastModified)
			if tt.wantOK {
				assert.False(t, r.cache.lastSuccess.IsZero())
			} else {
				assert.Equal(t, tt.seedCache.lastSuccess, r.cache.lastSuccess)
			}
		})
	}
}

// TestAttemptRepo2CPERefresh_RetryRecoversWithoutLosingCache calls the
// refresh helper directly (no timers) to verify a failed attempt keeps
// serving the last good mapping and a later success can recover.
func TestAttemptRepo2CPERefresh_RetryRecoversWithoutLosingCache(t *testing.T) {
	const mapping = `{"data":{"foo":{"cpes":["cpe:/o:foo"]}}}`
	hash := cpemapping.HashMapping([]byte(mapping))

	var fail atomic.Bool
	r := newRepo2CPE(newTestCentralClient(t, func(w http.ResponseWriter, _ *http.Request) {
		if fail.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(mapping))
	}))

	require.True(t, r.attemptRepo2CPERefresh(t.Context()))
	assert.Equal(t, hash, r.cache.hash)
	require.False(t, r.cache.lastSuccess.IsZero())
	lastGood := r.cache.lastSuccess

	fail.Store(true)
	require.False(t, r.attemptRepo2CPERefresh(t.Context()))
	assert.Equal(t, hash, r.cache.hash, "a failed refresh must not drop the last good mapping")
	assert.Equal(t, lastGood, r.cache.lastSuccess)

	fail.Store(false)
	require.True(t, r.attemptRepo2CPERefresh(t.Context()))
	assert.Equal(t, hash, r.cache.hash)
}

func TestFetchRepo2CPE(t *testing.T) {
	mapping := []byte(`{"data":{"foo":{"cpes":["cpe:/o:foo"]}}}`)
	hash := cpemapping.HashMapping(mapping)

	t.Run("never fetched returns ok=false", func(t *testing.T) {
		r := newRepo2CPE(&http.Client{})
		r.centralReachable.Store(false)

		gotMapping, gotHash, ok := r.FetchRepo2CPE(t.Context())

		assert.False(t, ok)
		assert.Nil(t, gotMapping)
		assert.Empty(t, gotHash)
	})

	t.Run("offline serves the last cached mapping with ok=true", func(t *testing.T) {
		r := newRepo2CPE(&http.Client{})
		r.cache = repo2CPECache{mapping: mapping, hash: hash, lastSuccess: time.Now()}
		r.centralReachable.Store(false)

		gotMapping, gotHash, ok := r.FetchRepo2CPE(t.Context())

		assert.True(t, ok)
		assert.Equal(t, mapping, gotMapping)
		assert.Equal(t, hash, gotHash)
	})

	t.Run("returned mapping is a copy of the cache", func(t *testing.T) {
		r := newRepo2CPE(&http.Client{})
		r.cache = repo2CPECache{mapping: slices.Clone(mapping), hash: hash, lastSuccess: time.Now()}
		r.centralReachable.Store(false)

		gotMapping, _, ok := r.FetchRepo2CPE(t.Context())
		require.True(t, ok)
		gotMapping[0] = 'X'
		assert.Equal(t, mapping, r.cache.mapping)
	})
}

func TestRepo2CPERun(t *testing.T) {
	tests := map[string]func(t *testing.T){
		"should fetch when a tick arrives and central is reachable": func(t *testing.T) {
			var fetches atomic.Int32
			r := newRepo2CPE(countingMappingClient(&fetches, testRepo2CPEMapping))
			r.centralReachable.Store(true)

			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(cancel)
			ticks := make(chan time.Time)
			go r.run(ctx, ticks)
			synctest.Wait()
			assert.Equal(t, int32(0), fetches.Load())

			ticks <- time.Time{}
			synctest.Wait()
			assert.Equal(t, int32(1), fetches.Load())
		},
		"should skip the fetch when central is unreachable": func(t *testing.T) {
			var fetches atomic.Int32
			r := newRepo2CPE(countingMappingClient(&fetches, testRepo2CPEMapping))
			r.centralReachable.Store(false)

			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(cancel)
			ticks := make(chan time.Time)
			go r.run(ctx, ticks)
			synctest.Wait()

			ticks <- time.Time{}
			synctest.Wait()
			assert.Equal(t, int32(0), fetches.Load())
		},
	}
	for name, fn := range tests {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, fn)
		})
	}
}

func TestRepo2CPEStart(t *testing.T) {
	tests := map[string]func(t *testing.T){
		"should fetch immediately on Start when central is reachable": func(t *testing.T) {
			var fetches atomic.Int32
			r := newRepo2CPE(countingMappingClient(&fetches, testRepo2CPEMapping))
			t.Cleanup(r.Stop)
			r.centralReachable.Store(true)

			require.NoError(t, r.Start())
			synctest.Wait()
			assert.Equal(t, int32(1), fetches.Load())
		},
		"should return an error on a second Start": func(t *testing.T) {
			var fetches atomic.Int32
			r := newRepo2CPE(countingMappingClient(&fetches, testRepo2CPEMapping))
			t.Cleanup(r.Stop)
			r.centralReachable.Store(true)

			require.NoError(t, r.Start())
			synctest.Wait()
			assert.Equal(t, errStartMoreThanOnce, r.Start())
			synctest.Wait()
			assert.Equal(t, int32(1), fetches.Load())
		},
		"should stop without further fetches": func(t *testing.T) {
			var fetches atomic.Int32
			r := newRepo2CPE(countingMappingClient(&fetches, testRepo2CPEMapping))
			t.Cleanup(r.Stop)
			r.centralReachable.Store(true)

			require.NoError(t, r.Start())
			synctest.Wait()
			assert.Equal(t, int32(1), fetches.Load())

			r.Stop()
			synctest.Wait()
			time.Sleep(repo2CPERefreshInterval)
			synctest.Wait()
			assert.Equal(t, int32(1), fetches.Load())
		},
		"should refetch after the success interval": func(t *testing.T) {
			var fetches atomic.Int32
			r := newRepo2CPE(countingMappingClient(&fetches, testRepo2CPEMapping))
			t.Cleanup(r.Stop)
			r.centralReachable.Store(true)

			require.NoError(t, r.Start())
			synctest.Wait()
			assert.Equal(t, int32(1), fetches.Load())

			time.Sleep(repo2CPERefreshInterval)
			synctest.Wait()
			assert.Equal(t, int32(2), fetches.Load())
		},
		"should refetch immediately when Notify kicks a running loop": func(t *testing.T) {
			var fetches atomic.Int32
			r := newRepo2CPE(countingMappingClient(&fetches, testRepo2CPEMapping))
			t.Cleanup(r.Stop)
			r.centralReachable.Store(true)

			require.NoError(t, r.Start())
			synctest.Wait()
			assert.Equal(t, int32(1), fetches.Load())

			r.Notify(common.SensorComponentEventCentralReachable)
			synctest.Wait()
			assert.Equal(t, int32(2), fetches.Load())
		},
	}
	for name, fn := range tests {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, fn)
		})
	}
}
