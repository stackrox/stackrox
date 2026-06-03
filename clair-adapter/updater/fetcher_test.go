package updater

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestZip(t *testing.T, files map[string][]byte) []byte {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	for name, data := range files {
		writer, err := zipWriter.Create(name)
		require.NoError(t, err)
		_, err = writer.Write(data)
		require.NoError(t, err)
	}

	require.NoError(t, zipWriter.Close())
	return buf.Bytes()
}

func TestFetcher_FetchOnce(t *testing.T) {
	zipData := createTestZip(t, map[string][]byte{
		"alpine.json.zst": []byte("alpine data"),
		"nvd.json.zst":    []byte("nvd data"),
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Last-Modified", "Mon, 02 Jan 2006 15:04:05 GMT")
		w.WriteHeader(http.StatusOK)
		w.Write(zipData)
	}))
	defer server.Close()

	updateServer := NewServer()
	fetcher := NewFetcher(updateServer, []string{server.URL})

	ctx := t.Context()
	err := fetcher.FetchOnce(ctx)
	require.NoError(t, err)

	// Verify bundles were loaded into server
	req := httptest.NewRequest(http.MethodGet, "/updater/alpine.json", nil)
	rec := httptest.NewRecorder()
	updateServer.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, []byte("alpine data"), rec.Body.Bytes())

	// Verify lastModified was set
	assert.Equal(t, "Mon, 02 Jan 2006 15:04:05 GMT", fetcher.lastModified)
}

func TestFetcher_FetchOnce_NotModified(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Mon, 02 Jan 2006 15:04:05 GMT", r.Header.Get("If-Modified-Since"))
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	updateServer := NewServer()
	fetcher := NewFetcher(updateServer, []string{server.URL})
	fetcher.lastModified = "Mon, 02 Jan 2006 15:04:05 GMT"

	ctx := t.Context()
	err := fetcher.FetchOnce(ctx)
	require.NoError(t, err)
}

func TestFetcher_FetchOnce_FallbackToNextURL(t *testing.T) {
	zipData := createTestZip(t, map[string][]byte{
		"alpine.json.zst": []byte("alpine data"),
	})

	// First server returns 404
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server1.Close()

	// Second server returns data
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Last-Modified", "Mon, 02 Jan 2006 15:04:05 GMT")
		w.WriteHeader(http.StatusOK)
		w.Write(zipData)
	}))
	defer server2.Close()

	updateServer := NewServer()
	fetcher := NewFetcher(updateServer, []string{server1.URL, server2.URL})

	ctx := t.Context()
	err := fetcher.FetchOnce(ctx)
	require.NoError(t, err)

	// Verify bundles were loaded from second server
	req := httptest.NewRequest(http.MethodGet, "/updater/alpine.json", nil)
	rec := httptest.NewRecorder()
	updateServer.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestFetcher_FetchOnce_AllURLsFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	updateServer := NewServer()
	fetcher := NewFetcher(updateServer, []string{server.URL})

	ctx := t.Context()
	err := fetcher.FetchOnce(ctx)
	require.Error(t, err)
}

func TestFetcher_Start(t *testing.T) {
	fetchCount := 0
	zipData := createTestZip(t, map[string][]byte{
		"alpine.json.zst": []byte("alpine data"),
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetchCount++
		w.Header().Set("Last-Modified", "Mon, 02 Jan 2006 15:04:05 GMT")
		w.WriteHeader(http.StatusOK)
		w.Write(zipData)
	}))
	defer server.Close()

	updateServer := NewServer()
	fetcher := NewFetcher(updateServer, []string{server.URL}, WithFetchInterval(10*time.Millisecond))

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	// Start fetcher in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- fetcher.Start(ctx)
	}()

	// Wait for at least 2 fetch cycles
	time.Sleep(25 * time.Millisecond)

	// Stop fetcher
	cancel()

	// Wait for fetcher to finish
	err := <-errChan
	require.NoError(t, err)

	// Should have fetched at least twice (immediate + at least one interval)
	assert.GreaterOrEqual(t, fetchCount, 2)
}

func TestFetcher_Options(t *testing.T) {
	updateServer := NewServer()

	t.Run("default options", func(t *testing.T) {
		f := NewFetcher(updateServer, []string{"http://example.com"})
		assert.Equal(t, 5*time.Minute, f.interval)
		assert.NotNil(t, f.client)
	})

	t.Run("custom interval", func(t *testing.T) {
		f := NewFetcher(updateServer, []string{"http://example.com"}, WithFetchInterval(10*time.Minute))
		assert.Equal(t, 10*time.Minute, f.interval)
	})

	t.Run("custom HTTP client", func(t *testing.T) {
		customClient := &http.Client{Timeout: 30 * time.Second}
		f := NewFetcher(updateServer, []string{"http://example.com"}, WithHTTPClient(customClient))
		assert.Equal(t, customClient, f.client)
	})
}

func TestFetcher_FetchOnce_InvalidZip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not a zip file"))
	}))
	defer server.Close()

	updateServer := NewServer()
	fetcher := NewFetcher(updateServer, []string{server.URL})

	ctx := t.Context()
	err := fetcher.FetchOnce(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unpack bundle")
}

func TestFetcher_FetchOnce_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	updateServer := NewServer()
	fetcher := NewFetcher(updateServer, []string{server.URL})

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // Cancel immediately

	err := fetcher.FetchOnce(ctx)
	require.Error(t, err)
}

func TestFetcher_FetchOnce_EmptyURLs(t *testing.T) {
	updateServer := NewServer()
	fetcher := NewFetcher(updateServer, []string{})

	ctx := t.Context()
	err := fetcher.FetchOnce(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no URLs configured")
}

func TestFetcher_FetchOnce_LastModifiedPersists(t *testing.T) {
	requestCount := 0
	zipData := createTestZip(t, map[string][]byte{
		"alpine.json.zst": []byte("alpine data"),
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			// First request: return data
			w.Header().Set("Last-Modified", "Mon, 02 Jan 2006 15:04:05 GMT")
			w.WriteHeader(http.StatusOK)
			w.Write(zipData)
		} else {
			// Subsequent requests: check If-Modified-Since header
			assert.Equal(t, "Mon, 02 Jan 2006 15:04:05 GMT", r.Header.Get("If-Modified-Since"))
			w.WriteHeader(http.StatusNotModified)
		}
	}))
	defer server.Close()

	updateServer := NewServer()
	fetcher := NewFetcher(updateServer, []string{server.URL})

	ctx := t.Context()

	// First fetch
	err := fetcher.FetchOnce(ctx)
	require.NoError(t, err)

	// Second fetch should send If-Modified-Since
	err = fetcher.FetchOnce(ctx)
	require.NoError(t, err)

	assert.Equal(t, 2, requestCount)
}
