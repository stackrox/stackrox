package datastore

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdaterSuccessfulDownload(t *testing.T) {
	bundleContent := `{"keys": [{"name": "test-key", "pem": "test"}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(bundleContent))
	}))
	defer server.Close()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "bundle.json")

	u := &keyBundleUpdater{
		client:   server.Client(),
		url:      server.URL,
		filePath: filePath,
		interval: time.Hour,
		stopSig:  concurrency.NewSignal(),
		doneSig:  concurrency.NewSignal(),
	}

	err := u.doDownload()
	require.NoError(t, err)

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, bundleContent, string(data))
}

func TestUpdaterHTTPErrorDoesNotModifyFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "bundle.json")
	originalContent := "original"
	require.NoError(t, os.WriteFile(filePath, []byte(originalContent), 0600))

	u := &keyBundleUpdater{
		client:   server.Client(),
		url:      server.URL,
		filePath: filePath,
		interval: time.Hour,
		stopSig:  concurrency.NewSignal(),
		doneSig:  concurrency.NewSignal(),
	}

	err := u.doDownload()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected HTTP status 500")

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, originalContent, string(data))
}

func TestUpdaterOversizedResponseRejected(t *testing.T) {
	largeBody := strings.Repeat("x", maxResponseBodySize+100)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(largeBody))
	}))
	defer server.Close()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "bundle.json")

	u := &keyBundleUpdater{
		client:   server.Client(),
		url:      server.URL,
		filePath: filePath,
		interval: time.Hour,
		stopSig:  concurrency.NewSignal(),
		doneSig:  concurrency.NewSignal(),
	}

	err := u.doDownload()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum size")
	assert.NoFileExists(t, filePath)
}

func TestUpdaterSequentialDownloads(t *testing.T) {
	content1 := `{"keys": [{"name": "key-v1", "pem": "test"}]}`
	content2 := `{"keys": [{"name": "key-v2", "pem": "test"}]}`

	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if callCount.Add(1) == 1 {
			_, _ = w.Write([]byte(content1))
		} else {
			_, _ = w.Write([]byte(content2))
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "bundle.json")

	u := &keyBundleUpdater{
		client:   server.Client(),
		url:      server.URL,
		filePath: filePath,
		interval: time.Hour,
		stopSig:  concurrency.NewSignal(),
		doneSig:  concurrency.NewSignal(),
	}

	require.NoError(t, u.doDownload())
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, content1, string(data))

	require.NoError(t, u.doDownload())
	data, err = os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, content2, string(data))
}

func TestUpdaterStopSignal(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "bundle.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("content"))
	}))
	defer server.Close()

	u := &keyBundleUpdater{
		client:   server.Client(),
		url:      server.URL,
		filePath: filePath,
		interval: 50 * time.Millisecond,
		stopSig:  concurrency.NewSignal(),
		doneSig:  concurrency.NewSignal(),
	}

	u.Start()

	// Poll for file existence instead of fixed sleep.
	require.Eventually(t, func() bool {
		_, err := os.Stat(filePath)
		return err == nil
	}, 2*time.Second, 10*time.Millisecond, "updater did not write the file")

	done := make(chan struct{})
	go func() {
		u.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("updater did not stop within timeout")
	}
}
