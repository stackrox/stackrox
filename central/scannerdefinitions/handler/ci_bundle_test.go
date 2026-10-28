package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	blob "github.com/stackrox/rox/central/blob/datastore"
	"github.com/stackrox/rox/central/scannerdefinitions/file"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type emptyBlobStore struct{ blob.Datastore }

func (emptyBlobStore) Get(context.Context, string, io.Writer) (*storage.Blob, bool, error) {
	return nil, false, nil
}

func TestCIMinimalBundleThroughCentral(t *testing.T) {
	data, err := os.ReadFile("../../../scanner/image/scanner/bundles/ci-minimal/vulnerabilities.zip")
	require.NoError(t, err)
	version := fmt.Sprintf("ci-minimal-%x", sha256.Sum256(data))
	modified := time.Date(2026, time.October, 16, 0, 0, 0, 0, time.UTC)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v4/vulnerability-bundles/"+version+"/vulnerabilities.zip", r.URL.Path)
		http.ServeContent(w, r, "vulnerabilities.zip", modified, bytes.NewReader(data))
	}))
	defer upstream.Close()
	oldURL, oldClient := scannerUpdateBaseURL, client
	scannerUpdateBaseURL, err = url.Parse(upstream.URL)
	require.NoError(t, err)
	client = upstream.Client()
	defer func() { scannerUpdateBaseURL, client = oldURL, oldClient }()

	root := t.TempDir()
	offline := newOfflineFileManager(emptyBlobStore{}, root)
	offline.Register(offlineScannerV4DefsBlobName)
	h := &httpHandler{
		online: true, updaterInterval: time.Hour,
		updaters: make(map[string]*requestedUpdater), dataDir: root, offlineFiles: offline,
	}
	defer func() {
		for _, u := range h.updaters {
			u.Stop()
		}
	}()
	req := httptest.NewRequest(http.MethodGet, "/api/extensions/scannerdefinitions?version="+version, nil)
	response := httptest.NewRecorder()
	h.ServeHTTP(response, req)
	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, data, response.Body.Bytes())
	assert.Equal(t, modified.Format(http.TimeFormat), response.Header().Get(lastModifiedHeader))
	u, err := h.getUpdater(vulnerabilityUpdaterType, version+"/vulnerabilities.zip")
	require.NoError(t, err)
	require.NoError(t, u.doUpdate(), "unchanged upstream returns 304")
	req.Header.Set(ifModifiedSinceHeader, modified.Format(http.TimeFormat))
	response = httptest.NewRecorder()
	h.ServeHTTP(response, req)
	assert.Equal(t, http.StatusNotModified, response.Code)
	assert.Empty(t, response.Body.Bytes())
}

func TestInvalidTimestampPreservesCachedDefinitions(t *testing.T) {
	for name, header := range map[string]string{"missing": "", "malformed": "invalid"} {
		t.Run(name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if header != "" {
					w.Header().Set(lastModifiedHeader, header)
				}
				_, _ = io.WriteString(w, "old data")
			}))
			defer upstream.Close()
			cached := file.New(filepath.Join(t.TempDir(), "definitions.zip"))
			modified := time.Date(2026, time.October, 20, 0, 0, 0, 0, time.UTC)
			require.NoError(t, cached.Write(bytes.NewBufferString("newer data"), modified))
			u := newUpdater(cached, upstream.Client(), upstream.URL, time.Hour)
			require.ErrorContains(t, u.doUpdate(), "modified time")
			f, gotTime, err := cached.Open()
			require.NoError(t, err)
			defer func() { require.NoError(t, f.Close()) }()
			got, err := io.ReadAll(f)
			require.NoError(t, err)
			assert.Equal(t, "newer data", string(got))
			assert.Equal(t, modified, gotTime)
		})
	}
}
