package updater

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_ServeBundleData(t *testing.T) {
	s := NewServer()

	bundles := []*BundleData{
		{
			Name:        "alpine",
			Data:        []byte("compressed alpine data"),
			Fingerprint: "abc123",
		},
		{
			Name:        "nvd",
			Data:        []byte("compressed nvd data"),
			Fingerprint: "def456",
		},
	}

	s.SetBundles(bundles)

	tests := map[string]struct {
		path           string
		wantStatusCode int
		wantData       []byte
		wantETag       string
	}{
		"get updater bundle": {
			path:           "/updater/alpine",
			wantStatusCode: http.StatusOK,
			wantData:       []byte("compressed alpine data"),
			wantETag:       "abc123",
		},
		"get enricher bundle": {
			path:           "/enricher/nvd",
			wantStatusCode: http.StatusOK,
			wantData:       []byte("compressed nvd data"),
			wantETag:       "def456",
		},
		"not found": {
			path:           "/updater/unknown",
			wantStatusCode: http.StatusNotFound,
		},
		"invalid path": {
			path:           "/invalid/path",
			wantStatusCode: http.StatusNotFound,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			s.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatusCode, rec.Code)

			if tt.wantStatusCode == http.StatusOK {
				assert.Equal(t, tt.wantData, rec.Body.Bytes())
				assert.Equal(t, tt.wantETag, rec.Header().Get("ETag"))
				assert.Equal(t, "application/zstd", rec.Header().Get("Content-Type"))
			}
		})
	}
}

func TestServer_NotModified(t *testing.T) {
	s := NewServer()

	s.SetBundles([]*BundleData{
		{
			Name:        "alpine",
			Data:        []byte("compressed alpine data"),
			Fingerprint: "abc123",
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/updater/alpine", nil)
	req.Header.Set("If-None-Match", "abc123")
	rec := httptest.NewRecorder()

	s.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotModified, rec.Code)
	assert.Empty(t, rec.Body.Bytes())
}

func TestServer_NotFound(t *testing.T) {
	s := NewServer()

	s.SetBundles([]*BundleData{
		{
			Name:        "alpine",
			Data:        []byte("compressed alpine data"),
			Fingerprint: "abc123",
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/updater/unknown", nil)
	rec := httptest.NewRecorder()

	s.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestServer_SetBundles_UpdatesExisting(t *testing.T) {
	s := NewServer()

	// Set initial bundles
	s.SetBundles([]*BundleData{
		{
			Name:        "alpine",
			Data:        []byte("old data"),
			Fingerprint: "old",
		},
	})

	// Update with new bundles
	s.SetBundles([]*BundleData{
		{
			Name:        "alpine",
			Data:        []byte("new data"),
			Fingerprint: "new",
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/updater/alpine", nil)
	rec := httptest.NewRecorder()

	s.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, []byte("new data"), rec.Body.Bytes())
	assert.Equal(t, "new", rec.Header().Get("ETag"))
}
