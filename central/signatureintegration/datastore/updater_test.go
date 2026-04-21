package datastore

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
	u, err := newUpdater(&http.Client{Timeout: 5 * time.Second}, manifestURL, targetDir, time.Second, nil)
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
		{"https://example.com/keys/manifest.json", "  key.pub \n", "https://example.com/keys/key.pub", false},
		{"https://example.com/keys/manifest.json", "sub/key.pub", "https://example.com/keys/sub/key.pub", false},
		{"https://example.com/manifest.json", "key.pub", "https://example.com/key.pub", false},
		{"https://example.com/keys/manifest.json", "https://example.com/keys/", "", true},
		{"https://example.com/keys/manifest.json", "keys/", "", true},
		{"://invalid", "key.pub", "", true},
		{"https://example.com/keys/manifest.json", "file:///etc/shadow", "", true},
		{"https://example.com/keys/manifest.json", "gopher://evil.com/key", "", true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.manifestURL, tt.keyURL), func(t *testing.T) {
			got, err := resolveKeyURL(tt.manifestURL, tt.keyURL)
			if tt.wantErr {
				require.Errorf(t, err, "manifest=%q key=%q", tt.manifestURL, tt.keyURL)
				return
			}
			require.NoErrorf(t, err, "manifest=%q key=%q", tt.manifestURL, tt.keyURL)
			require.Equalf(t, tt.want, got, "manifest=%q key=%q", tt.manifestURL, tt.keyURL)
		})
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
	_, err := newUpdater(nil, "https://example.com/manifest.json", t.TempDir(), time.Second, nil)
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

func TestUpdateFailsOnEmptyManifest(t *testing.T) {
	targetDir := t.TempDir()
	manifest := manifest{Keys: []manifestKey{}}
	manifestBody, err := json.Marshal(manifest)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(manifestBody)
	}))
	defer server.Close()

	u := newTestUpdater(t, server.URL+"/manifest.json", targetDir)
	require.Error(t, u.update())
}

func TestUpdateFailsOnInvalidManifestJSON(t *testing.T) {
	targetDir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json"))
	}))
	defer server.Close()

	u := newTestUpdater(t, server.URL+"/manifest.json", targetDir)
	require.Error(t, u.update())
}

func TestUpdateFailsOnManifestHTTPError(t *testing.T) {
	targetDir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	u := newTestUpdater(t, server.URL+"/manifest.json", targetDir)
	require.Error(t, u.update())
}

func TestReplaceDirectoryContentsIsAtomic(t *testing.T) {
	targetDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(targetDir, "old.txt"), []byte("old"), 0o600))

	sourceDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "new.txt"), []byte("new"), 0o600))

	require.NoError(t, replaceDirectoryContents(targetDir, sourceDir))

	content, err := os.ReadFile(filepath.Join(targetDir, "new.txt"))
	require.NoError(t, err)
	require.Equal(t, "new", string(content))

	_, err = os.Stat(filepath.Join(targetDir, "old.txt"))
	require.True(t, os.IsNotExist(err))
}

func TestStartStopLifecycle(t *testing.T) {
	pem := validPEM(t)
	targetDir := t.TempDir()
	manifest := manifest{
		Keys: []manifestKey{{Name: "key.pub", URL: "key.pub"}},
	}
	manifestBody, err := json.Marshal(manifest)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manifest.json":
			_, _ = w.Write(manifestBody)
		case "/key.pub":
			_, _ = w.Write([]byte(pem))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	u := newTestUpdater(t, server.URL+"/manifest.json", targetDir)
	u.Start()
	u.Start() // second call should be no-op (sync.Once)
	u.Stop()
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

func TestNewUpdaterRejectsZeroInterval(t *testing.T) {
	_, err := newUpdater(&http.Client{}, "https://example.com/manifest.json", t.TempDir(), 0, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "interval must be positive")
}

func TestDoUpdate_OnSuccessCalledOnSuccess(t *testing.T) {
	pem := validPEM(t)
	targetDir := t.TempDir()

	manifest := manifest{
		Keys: []manifestKey{{Name: "key.pub", URL: "key.pub"}},
	}
	manifestBody, err := json.Marshal(manifest)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manifest.json":
			_, _ = w.Write(manifestBody)
		case "/key.pub":
			_, _ = w.Write([]byte(pem))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	called := make(chan struct{}, 1)
	u, err := newUpdater(
		&http.Client{Timeout: 5 * time.Second},
		server.URL+"/manifest.json",
		targetDir,
		time.Second,
		func() { called <- struct{}{} },
	)
	require.NoError(t, err)

	u.doUpdate()

	select {
	case <-called:
		// expected
	default:
		t.Fatal("onSuccess was not called after a successful update")
	}
}

func TestDoUpdate_OnSuccessNotCalledOnFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	called := false
	u, err := newUpdater(
		&http.Client{Timeout: 5 * time.Second},
		server.URL+"/manifest.json",
		t.TempDir(),
		time.Second,
		func() { called = true },
	)
	require.NoError(t, err)

	u.doUpdate()

	require.False(t, called, "onSuccess must not be called when the update fails")
}

func TestResolveKeyRefsRejectsEmptyName(t *testing.T) {
	_, err := resolveKeyRefsFromManifest("https://example.com/manifest.json", manifest{
		Keys: []manifestKey{{Name: "", URL: "key.pub"}},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty name")
}

func TestResolveKeyRefsRejectsPathSeparatorInName(t *testing.T) {
	_, err := resolveKeyRefsFromManifest("https://example.com/manifest.json", manifest{
		Keys: []manifestKey{{Name: "../etc/evil", URL: "key.pub"}},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "path separator")
}

func TestDownloadBytesRejectsOversizedResponse(t *testing.T) {
	// Serve a response that exceeds maxResponseBodySize.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Write maxResponseBodySize + 100 bytes.
		_, _ = w.Write([]byte(strings.Repeat("x", maxResponseBodySize+100)))
	}))
	defer server.Close()

	u := newTestUpdater(t, server.URL+"/manifest.json", t.TempDir())
	_, err := u.downloadBytes(server.URL + "/big-file")
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeds maximum size")
}
