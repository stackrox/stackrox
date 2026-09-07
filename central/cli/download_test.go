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

func writeTarball(t *testing.T, dir, filename string, content []byte) {
	t.Helper()
	f, err := os.Create(filepath.Join(dir, filename+".tar.gz"))
	require.NoError(t, err)
	defer func() { require.NoError(t, f.Close()) }()

	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: filename,
		Size: int64(len(content)),
		Mode: 0755,
	}))
	_, err = tw.Write(content)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
}

func TestServeFromDir(t *testing.T) {
	dir := t.TempDir()
	const binary = "roxctl-linux-amd64"
	payload := []byte("fake roxctl binary content")
	writeTarball(t, dir, binary, payload)

	t.Run("serves binary extracted from tarball", func(t *testing.T) {
		rr := httptest.NewRecorder()
		require.NoError(t, serveFromDir(rr, dir, binary))
		assert.Equal(t, `attachment; filename="roxctl-linux-amd64"`, rr.Header().Get("Content-Disposition"))
		assert.Equal(t, "application/octet-stream", rr.Header().Get("Content-Type"))
		assert.Equal(t, "26", rr.Header().Get("Content-Length"))
		assert.Equal(t, payload, rr.Body.Bytes())
	})

	t.Run("returns error for missing tarball", func(t *testing.T) {
		rr := httptest.NewRecorder()
		assert.Error(t, serveFromDir(rr, dir, "roxctl-darwin-amd64"))
	})
}

func TestHandler(t *testing.T) {
	dir := t.TempDir()
	orig := downloadPath
	downloadPath = dir
	t.Cleanup(func() { downloadPath = orig })

	payload := []byte("fake roxctl binary content")
	writeTarball(t, dir, "roxctl-linux-amd64", payload)

	t.Run("serves binary via HTTP handler", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/cli/download/roxctl-linux-amd64", nil)
		rr := httptest.NewRecorder()
		Handler()(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, payload, rr.Body.Bytes())
	})

	t.Run("returns 404 for missing binary", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/cli/download/roxctl-darwin-amd64", nil)
		rr := httptest.NewRecorder()
		Handler()(rr, req)
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}
