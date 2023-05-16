package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustGetRequest(t *testing.T) *http.Request {
	centralURL := "https://central.stackrox.svc/scannerdefinitions?uuid=e799c68a-671f-44db-9682-f24248cd0ffe"
	req, err := http.NewRequest(http.MethodGet, centralURL, nil)
	require.NoError(t, err)

	return req
}

func mustGetRequestWithFile(t *testing.T, file string) *http.Request {
	centralURL := fmt.Sprintf("https://central.stackrox.svc/scannerdefinitions?uuid=e799c68a-671f-44db-9682-f24248cd0ffe&file=%s", file)
	req, err := http.NewRequest(http.MethodGet, centralURL, nil)
	require.NoError(t, err)

	return req
}

func mustGetBadRequest(t *testing.T) *http.Request {
	centralURL := "https://central.stackrox.svc/scannerdefinitions?uuid=fail"
	req, err := http.NewRequest(http.MethodGet, centralURL, nil)
	require.NoError(t, err)

	return req
}

func TestServeHTTP_Offline_Get(t *testing.T) {
	t.Setenv(env.OfflineModeEnv.EnvVar(), "true")

	tmpDir := t.TempDir()
	h := New(nil, handlerOpts{
		offlineVulnDefsDir: tmpDir,
	})

	// No scanner defs found.
	req := mustGetRequest(t)
	w := mock.NewResponseWriter()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Add scanner defs.
	f, err := os.Create(filepath.Join(tmpDir, offlineScannerDefsName))
	require.NoError(t, err)
	_, err = f.Write([]byte("Hello, World!"))
	require.NoError(t, err)

	w.Data.Reset()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Hello, World!", w.Data.String())
}

func TestServeHTTP_Online_Get(t *testing.T) {
	tmpDir := t.TempDir()
	h := New(nil, handlerOpts{
		offlineVulnDefsDir: tmpDir,
	})

	w := mock.NewResponseWriter()

	// Should not get anything.
	req := mustGetBadRequest(t)
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Should get file from online update.
	req = mustGetRequestWithFile(t, "manifest.json")
	w.Data.Reset()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Regexpf(t, `{"since":".*","until":".*"}`, w.Data.String(), "content did not match")
	// Should get online update.
	req = mustGetRequest(t)
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Write offline definitions.
	f, err := os.Create(filepath.Join(tmpDir, offlineScannerDefsName))
	require.NoError(t, err)
	_, err = f.Write([]byte("Hello, World!"))
	require.NoError(t, err)

	// Set the offline dump's modified time to later than the online update's.
	handler := h.(*httpHandler)
	mustSetModTime(t, handler.offlineFile.Path(), time.Now().Add(time.Minute))

	// Served the offline dump, as it is more recent.
	w.Data.Reset()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Hello, World!", w.Data.String())

	// Set the offline dump's modified time to earlier than the online update's.
	mustSetModTime(t, handler.offlineFile.Path(), nov23)

	// Serve the online dump, as it is now more recent.
	w.Data.Reset()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEqual(t, "Hello, World!", w.Data.String())

	// File is unmodified.
	req.Header.Set(ifModifiedSinceHeader, time.Now().UTC().Format(http.TimeFormat))
	w.Data.Reset()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotModified, w.Code)
	assert.Empty(t, w.Data.String())
}

func mustSetModTime(t *testing.T, path string, modTime time.Time) {
	require.NoError(t, os.Chtimes(path, time.Now(), modTime))
}
