package datastore

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func validPEM(t *testing.T) string {
	t.Helper()
	return goodCosignConfig.GetPublicKeys()[0].GetPublicKeyPemEnc()
}

func newTestUpdater(t *testing.T, manifestURL, targetDir string) *updater {
	t.Helper()
	u, err := newUpdater(&http.Client{Timeout: 5 * time.Second}, manifestURL, targetDir, time.Second)
	require.NoError(t, err)
	return u
}

func TestResolveKeyURL(t *testing.T) {
	tests := []struct {
		manifestURL string
		keyURL      string
		want        string
		wantErr     bool
	}{
		{"https://a.com/dir/manifest.json", "https://b.com/key.pub", "https://b.com/key.pub", false},
		{"https://a.com/dir/manifest.json", "http://b.com/key.pub", "http://b.com/key.pub", false},
		{"https://example.com/keys/manifest.json", "release-key.pub", "https://example.com/keys/release-key.pub", false},
		{"https://example.com/keys/manifest.json", "sub/key.pub", "https://example.com/keys/sub/key.pub", false},
		{"https://example.com/manifest.json", "key.pub", "https://example.com/key.pub", false},
		{"https://example.com/keys/manifest.json", "https://example.com/keys/", "", true},
		{"https://example.com/keys/manifest.json", "keys/", "", true},
		{"://invalid", "key.pub", "", true},
	}

	for _, tt := range tests {
		got, err := resolveKeyURL(tt.manifestURL, tt.keyURL)
		if tt.wantErr {
			require.Error(t, err, "manifest=%q key=%q", tt.manifestURL, tt.keyURL)
			continue
		}
		require.NoError(t, err, "manifest=%q key=%q", tt.manifestURL, tt.keyURL)
		require.Equal(t, tt.want, got, "manifest=%q key=%q", tt.manifestURL, tt.keyURL)
	}
}

func TestUpdateDownloadsAllFiles(t *testing.T) {
	pem := validPEM(t)
	targetDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(targetDir, "stale.pub"), []byte("stale"), 0o600))

	manifest := manifest{
		Keys: []manifestKey{
			{Name: "key-a.pub", URL: "key-a.pub"},
			{Name: "key-b.pub", URL: "nested/key-b.pub"},
		},
	}
	manifestBody, err := json.Marshal(manifest)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manifest.json":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(manifestBody)
		case "/key-a.pub", "/nested/key-b.pub":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(pem))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	u := newTestUpdater(t, server.URL+"/manifest.json", targetDir)
	require.NoError(t, u.update())

	entries, err := os.ReadDir(targetDir)
	require.NoError(t, err)
	require.Len(t, entries, 2)

	a, err := os.ReadFile(filepath.Join(targetDir, "key-a.pub"))
	require.NoError(t, err)
	require.Equal(t, pem, string(a))

	b, err := os.ReadFile(filepath.Join(targetDir, "key-b.pub"))
	require.NoError(t, err)
	require.Equal(t, pem, string(b))

	_, err = os.Stat(filepath.Join(targetDir, "stale.pub"))
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))
}

func TestNewUpdaterRequiresClient(t *testing.T) {
	_, err := newUpdater(nil, "https://example.com/manifest.json", t.TempDir(), time.Second)
	require.Error(t, err)
}

func TestUpdateAllowsPartialDownloads(t *testing.T) {
	pem := validPEM(t)
	targetDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(targetDir, "stale.pub"), []byte("stale"), 0o600))

	manifest := manifest{
		Keys: []manifestKey{
			{Name: "key-a.pub", URL: "key-a.pub"},
			{Name: "missing.pub", URL: "missing.pub"},
		},
	}
	manifestBody, err := json.Marshal(manifest)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manifest.json":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(manifestBody)
		case "/key-a.pub":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(pem))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	u := newTestUpdater(t, server.URL+"/manifest.json", targetDir)
	require.NoError(t, u.update())

	entries, err := os.ReadDir(targetDir)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	a, err := os.ReadFile(filepath.Join(targetDir, "key-a.pub"))
	require.NoError(t, err)
	require.Equal(t, pem, string(a))

	_, err = os.Stat(filepath.Join(targetDir, "stale.pub"))
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))
}

func TestUpdateAllowsDuplicateOutputFileNames_LastWriteWins(t *testing.T) {
	pem := validPEM(t)
	targetDir := t.TempDir()

	manifest := manifest{
		Keys: []manifestKey{
			{Name: "key.pub", URL: "keys/key.pub"},
			{Name: "key.pub", URL: "other/key.pub"},
		},
	}
	manifestBody, err := json.Marshal(manifest)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manifest.json":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(manifestBody)
		case "/keys/key.pub", "/other/key.pub":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(pem))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	u := newTestUpdater(t, server.URL+"/manifest.json", targetDir)
	require.NoError(t, u.update())
	content, err := os.ReadFile(filepath.Join(targetDir, "key.pub"))
	require.NoError(t, err)
	require.Equal(t, pem, string(content))
}

func TestUpdateFailsWhenNoFilesCanBeDownloaded(t *testing.T) {
	targetDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(targetDir, "existing.pub"), []byte("existing"), 0o600))
	manifest := manifest{
		Keys: []manifestKey{
			{Name: "Missing A", URL: "a.pub"},
			{Name: "Missing B", URL: "b.pub"},
		},
	}
	manifestBody, err := json.Marshal(manifest)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/manifest.json" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(manifestBody)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	u := newTestUpdater(t, server.URL+"/manifest.json", targetDir)
	require.Error(t, u.update())

	existing, err := os.ReadFile(filepath.Join(targetDir, "existing.pub"))
	require.NoError(t, err)
	require.Equal(t, "existing", string(existing))
}
