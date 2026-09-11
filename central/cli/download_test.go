package cli

import (
	"archive/tar"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTarball(t *testing.T, dir, tarName, entryName string, content []byte) {
	t.Helper()
	f, err := os.Create(filepath.Join(dir, tarName+".tar.gz"))
	require.NoError(t, err)

	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: entryName,
		Size: int64(len(content)),
		Mode: 0755,
	}))
	_, err = tw.Write(content)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	require.NoError(t, f.Close())
}

func TestHandler(t *testing.T) {
	dir := t.TempDir()
	handler := handlerWithDir(dir)
	payload := []byte("fake roxctl binary content")
	writeTarball(t, dir, "roxctl-linux-amd64", "roxctl-linux-amd64", payload)
	// Windows: tarball strips .exe from the name (required by release pipeline),
	// but the entry inside the tarball preserves .exe.
	writeTarball(t, dir, "roxctl-windows-amd64", "roxctl-windows-amd64.exe", payload)

	t.Run("serves linux binary via GET", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/cli/download/roxctl-linux-amd64", nil)
		rr := httptest.NewRecorder()
		handler(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, `attachment; filename="roxctl-linux-amd64"`, rr.Header().Get("Content-Disposition"))
		assert.Equal(t, "application/octet-stream", rr.Header().Get("Content-Type"))
		assert.Equal(t, "26", rr.Header().Get("Content-Length"))
		assert.Equal(t, "none", rr.Header().Get("Accept-Ranges"))
		assert.Equal(t, "no-cache", rr.Header().Get("Cache-Control"))
		assert.Equal(t, payload, rr.Body.Bytes())
	})

	t.Run("serves windows binary via GET", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/cli/download/roxctl-windows-amd64.exe", nil)
		rr := httptest.NewRecorder()
		handler(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, `attachment; filename="roxctl-windows-amd64.exe"`, rr.Header().Get("Content-Disposition"))
		assert.Equal(t, payload, rr.Body.Bytes())
	})

	t.Run("returns headers but no body for HEAD", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodHead, "/api/cli/download/roxctl-linux-amd64", nil)
		rr := httptest.NewRecorder()
		handler(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "26", rr.Header().Get("Content-Length"))
		assert.Empty(t, rr.Body.Bytes())
	})

	t.Run("returns 404 for missing binary", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/cli/download/roxctl-darwin-amd64", nil)
		rr := httptest.NewRecorder()
		handler(rr, req)
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("returns 405 for unsupported method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/cli/download/roxctl-linux-amd64", nil)
		rr := httptest.NewRecorder()
		handler(rr, req)
		assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
		assert.Equal(t, "GET, HEAD", rr.Header().Get("Allow"))
	})
}
