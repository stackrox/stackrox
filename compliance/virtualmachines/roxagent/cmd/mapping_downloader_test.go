package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/stackrox/rox/pkg/filedownloader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// roundTripperFunc adapts a function to http.RoundTripper. It is a local
// copy of the identical helper in pkg/filedownloader/filedownloader_test.go:
// both packages need it to fake HTTP calls inside a synctest bubble without
// real network I/O, and it's small enough that sharing it isn't worth an
// exported, test-only symbol in either package.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestNewMappingDownloader(t *testing.T) {
	t.Run("should publish the response body to cachePath", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"data":{}}`))
		}))
		defer srv.Close()

		cachePath := filepath.Join(t.TempDir(), "repo2cpe.json")
		d := newMappingDownloader(srv.URL, cachePath)

		require.NoError(t, d.DownloadOnce(t.Context()))

		got, err := os.ReadFile(cachePath)
		require.NoError(t, err)
		assert.Equal(t, `{"data":{}}`, string(got))
	})

	t.Run("should leave any previously cached file untouched on a failed fetch", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		cachePath := filepath.Join(t.TempDir(), "repo2cpe.json")
		require.NoError(t, os.WriteFile(cachePath, []byte("stale"), 0o600))

		// filedownloader.New with a single, non-retrying attempt, rather than
		// newMappingDownloader (which always retries 3 times with backoff): this
		// test verifies atomic-write safety on failure, not retry behavior -
		// already covered, deterministically, by the next subtest - so it has
		// no reason to sit through a real multi-second backoff.
		d := filedownloader.New(srv.URL, cachePath, mappingRefreshInterval,
			filedownloader.WithHTTPClient(&http.Client{}),
		)

		require.Error(t, d.DownloadOnce(t.Context()))

		got, err := os.ReadFile(cachePath)
		require.NoError(t, err)
		assert.Equal(t, "stale", string(got))
	})

	t.Run("should retry mappingFetchMaxAttempts times on persistent failure", func(t *testing.T) {
		// Exercises the mappingFetchMaxAttempts/mappingFetchBaseBackoff wiring
		// under fake time via a fake http.RoundTripper, not newMappingDownloader
		// itself: newMappingDownloader always builds a real proxy.RoundTripper()
		// client, and driving real network I/O inside a synctest bubble would be
		// fragile - it only works because real I/O happens not to block on fake
		// time today, which isn't guaranteed to stay true. filedownloader.New
		// with the same retry constants and a roundTripperFunc gives the same
		// coverage deterministically.
		synctest.Test(t, func(t *testing.T) {
			var calls atomic.Int32
			client := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}, nil
			})}

			cachePath := filepath.Join(t.TempDir(), "repo2cpe.json")
			d := filedownloader.New("http://mapping.invalid/repo2cpe.json", cachePath, mappingRefreshInterval,
				filedownloader.WithHTTPClient(client),
				filedownloader.WithRetryPolicy(mappingFetchMaxAttempts, mappingFetchBaseBackoff),
			)

			require.Error(t, d.DownloadOnce(t.Context()))
			assert.Equal(t, int32(mappingFetchMaxAttempts), calls.Load())
		})
	})
}
