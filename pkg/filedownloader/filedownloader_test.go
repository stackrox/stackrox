package filedownloader

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuccessfulDownload(t *testing.T) {
	content := `{"keys": [{"name": "test-key"}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(content))
	}))
	defer server.Close()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "data.json")

	d := New(server.URL, filePath, time.Hour, WithHTTPClient(server.Client()))
	require.NoError(t, d.doDownload(t.Context()))

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestHTTPErrorDoesNotModifyFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "data.json")
	original := "original"
	require.NoError(t, os.WriteFile(filePath, []byte(original), 0600))

	d := New(server.URL, filePath, time.Hour, WithHTTPClient(server.Client()))
	err := d.doDownload(t.Context())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected HTTP status 500 Internal Server Error")

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, original, string(data))
}

func TestOversizedResponseRejected(t *testing.T) {
	largeBody := strings.Repeat("x", defaultMaxSize+100)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(largeBody))
	}))
	defer server.Close()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "data.json")

	d := New(server.URL, filePath, time.Hour, WithHTTPClient(server.Client()))
	err := d.doDownload(t.Context())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum size")
	assert.NoFileExists(t, filePath)
}

func TestSequentialDownloads(t *testing.T) {
	v1 := `{"version": 1}`
	v2 := `{"version": 2}`

	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if callCount.Add(1) == 1 {
			_, _ = w.Write([]byte(v1))
		} else {
			_, _ = w.Write([]byte(v2))
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "data.json")

	d := New(server.URL, filePath, time.Hour, WithHTTPClient(server.Client()))

	require.NoError(t, d.doDownload(t.Context()))
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, v1, string(data))

	require.NoError(t, d.doDownload(t.Context()))
	data, err = os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, v2, string(data))
}

func TestStopSignal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("content"))
	}))
	defer server.Close()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "data.json")

	d := New(server.URL, filePath, 50*time.Millisecond, WithHTTPClient(server.Client()))
	d.Start()

	require.Eventually(t, func() bool {
		_, err := os.Stat(filePath)
		return err == nil
	}, 2*time.Second, 50*time.Millisecond, "downloader did not write the file")

	done := make(chan struct{})
	go func() { d.Stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("downloader did not stop within timeout")
	}
}

func TestOnCompleteCallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "data.json")

	var gotErr error
	var gotDuration time.Duration
	d := New(server.URL, filePath, time.Hour,
		WithHTTPClient(server.Client()),
		WithOnComplete(func(err error, dur time.Duration) {
			gotErr = err
			gotDuration = dur
		}),
	)

	d.download(t.Context())
	assert.NoError(t, gotErr)
	assert.Greater(t, gotDuration, time.Duration(0))
}

func TestOnCompleteCallbackOnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "data.json")

	var gotErr error
	d := New(server.URL, filePath, time.Hour,
		WithHTTPClient(server.Client()),
		WithOnComplete(func(err error, _ time.Duration) {
			gotErr = err
		}),
	)

	d.download(t.Context())
	assert.Error(t, gotErr)
	assert.Contains(t, gotErr.Error(), "503")
}

func TestClampsInterval(t *testing.T) {
	d := New("http://example.com", "/tmp/test", time.Millisecond)
	assert.GreaterOrEqual(t, d.interval, minInterval)

	d = New("http://example.com", "/tmp/test", minInterval)
	assert.Equal(t, minInterval, d.interval)

	long := 2 * minInterval
	d = New("http://example.com", "/tmp/test", long)
	assert.Equal(t, long, d.interval)
}

func TestCustomMaxSize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("too big"))
	}))
	defer server.Close()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "data.json")

	d := New(server.URL, filePath, time.Hour,
		WithHTTPClient(server.Client()),
		WithMaxSize(3),
	)
	err := d.doDownload(t.Context())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum size")
}

func TestAtomicWriteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.json")

	require.NoError(t, atomicWriteFile(path, []byte("hello")))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}
