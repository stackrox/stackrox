package indexer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stackrox/rox/clair-adapter/clairclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndexer_IndexContainerImage(t *testing.T) {
	tests := map[string]struct {
		hashID       string
		imageURL     string
		mockResponse *clairclient.IndexReport
		mockStatus   int
		expectError  bool
	}{
		"successful indexing": {
			hashID:   "sha256:abc123",
			imageURL: "registry.io/image:tag",
			mockResponse: &clairclient.IndexReport{
				ManifestHash: "sha256:abc123",
				State:        "IndexFinished",
				Success:      true,
				Packages:     map[string]clairclient.Package{},
			},
			mockStatus:  http.StatusCreated,
			expectError: false,
		},
		"clair returns error": {
			hashID:      "sha256:def456",
			imageURL:    "registry.io/image:tag",
			mockStatus:  http.StatusInternalServerError,
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create mock Clair server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/indexer/api/v1/index_report", r.URL.Path)

				// Verify request body
				var manifest clairclient.Manifest
				err := json.NewDecoder(r.Body).Decode(&manifest)
				require.NoError(t, err)
				assert.Equal(t, tc.hashID, manifest.Hash)
				assert.Empty(t, manifest.Layers) // Stub fetcher returns empty layers

				w.WriteHeader(tc.mockStatus)
				if tc.mockResponse != nil {
					err := json.NewEncoder(w).Encode(tc.mockResponse)
					require.NoError(t, err)
				}
			}))
			defer server.Close()

			// Create client and indexer with stub layer fetcher
			client, err := clairclient.NewClient(server.URL)
			require.NoError(t, err)

			// Stub fetcher that returns empty layers
			stubFetcher := func(ctx context.Context, imageURL string, opts indexOpts) ([]clairclient.Layer, error) {
				return []clairclient.Layer{}, nil
			}

			indexer := NewLocalIndexer(client, nil, WithLayerFetcher(stubFetcher))

			// Execute test
			ctx := context.Background()
			report, err := indexer.IndexContainerImage(ctx, tc.hashID, tc.imageURL)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, report)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, report)
				assert.Equal(t, tc.hashID, report.ManifestHash)
				assert.True(t, report.Success)
			}
		})
	}
}

func TestIndexer_IndexContainerImage_WithMetadataStore(t *testing.T) {
	mockReport := &clairclient.IndexReport{
		ManifestHash: "sha256:stored123",
		State:        "IndexFinished",
		Success:      true,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		err := json.NewEncoder(w).Encode(mockReport)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := clairclient.NewClient(server.URL)
	require.NoError(t, err)

	// Mock metadata store
	mockStore := &mockIndexerMetadataStore{
		stored: make(map[string]time.Time),
	}

	// Stub fetcher that returns empty layers
	stubFetcher := func(ctx context.Context, imageURL string, opts indexOpts) ([]clairclient.Layer, error) {
		return []clairclient.Layer{}, nil
	}

	indexer := NewLocalIndexer(client, mockStore, WithLayerFetcher(stubFetcher))

	ctx := context.Background()
	report, err := indexer.IndexContainerImage(ctx, "sha256:stored123", "registry.io/image:v1")

	require.NoError(t, err)
	assert.NotNil(t, report)

	// Verify metadata was stored
	assert.Contains(t, mockStore.stored, "sha256:stored123")
	assert.True(t, mockStore.stored["sha256:stored123"].After(time.Now()))
}

func TestIndexer_GetIndexReport(t *testing.T) {
	tests := map[string]struct {
		hashID       string
		mockResponse *clairclient.IndexReport
		mockStatus   int
		expectFound  bool
		expectError  bool
	}{
		"report found": {
			hashID: "sha256:exists",
			mockResponse: &clairclient.IndexReport{
				ManifestHash: "sha256:exists",
				State:        "IndexFinished",
				Success:      true,
			},
			mockStatus:  http.StatusOK,
			expectFound: true,
		},
		"report not found": {
			hashID:      "sha256:notfound",
			mockStatus:  http.StatusNotFound,
			expectFound: false,
		},
		"server error": {
			hashID:      "sha256:error",
			mockStatus:  http.StatusInternalServerError,
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Contains(t, r.URL.Path, tc.hashID)

				w.WriteHeader(tc.mockStatus)
				if tc.mockResponse != nil {
					err := json.NewEncoder(w).Encode(tc.mockResponse)
					require.NoError(t, err)
				}
			}))
			defer server.Close()

			client, err := clairclient.NewClient(server.URL)
			require.NoError(t, err)

			indexer := NewLocalIndexer(client, nil)

			ctx := context.Background()
			report, found, err := indexer.GetIndexReport(ctx, tc.hashID)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectFound, found)
				if tc.expectFound {
					assert.NotNil(t, report)
					assert.Equal(t, tc.hashID, report.ManifestHash)
				} else {
					assert.Nil(t, report)
				}
			}
		})
	}
}

func TestIndexer_HasIndexReport(t *testing.T) {
	tests := map[string]struct {
		hashID      string
		mockStatus  int
		expectFound bool
		expectError bool
	}{
		"report exists": {
			hashID:      "sha256:exists",
			mockStatus:  http.StatusOK,
			expectFound: true,
		},
		"report does not exist": {
			hashID:      "sha256:notfound",
			mockStatus:  http.StatusNotFound,
			expectFound: false,
		},
		"server error": {
			hashID:      "sha256:error",
			mockStatus:  http.StatusInternalServerError,
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.mockStatus)
				if tc.mockStatus == http.StatusOK {
					report := &clairclient.IndexReport{ManifestHash: tc.hashID}
					err := json.NewEncoder(w).Encode(report)
					require.NoError(t, err)
				}
			}))
			defer server.Close()

			client, err := clairclient.NewClient(server.URL)
			require.NoError(t, err)

			indexer := NewLocalIndexer(client, nil)

			ctx := context.Background()
			exists, err := indexer.HasIndexReport(ctx, tc.hashID)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectFound, exists)
			}
		})
	}
}

func TestIndexer_WithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		report := &clairclient.IndexReport{ManifestHash: "sha256:test"}
		err := json.NewEncoder(w).Encode(report)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := clairclient.NewClient(server.URL)
	require.NoError(t, err)

	// Stub fetcher that returns empty layers
	stubFetcher := func(ctx context.Context, imageURL string, opts indexOpts) ([]clairclient.Layer, error) {
		return []clairclient.Layer{}, nil
	}

	indexer := NewLocalIndexer(client, nil, WithLayerFetcher(stubFetcher))

	ctx := context.Background()
	_, err = indexer.IndexContainerImage(
		ctx,
		"sha256:test",
		"registry.io/image:v1",
		WithBasicAuth("user", "pass"),
		WithInsecureSkipTLSVerify(true),
	)

	require.NoError(t, err)
}

// mockIndexerMetadataStore is a simple in-memory implementation for testing
type mockIndexerMetadataStore struct {
	stored map[string]time.Time
}

func (m *mockIndexerMetadataStore) StoreManifest(ctx context.Context, manifestID string, expiration time.Time) error {
	m.stored[manifestID] = expiration
	return nil
}

func (m *mockIndexerMetadataStore) ManifestExists(ctx context.Context, manifestID string) (bool, error) {
	_, exists := m.stored[manifestID]
	return exists, nil
}

func (m *mockIndexerMetadataStore) GCManifests(ctx context.Context, expiration time.Time, limit int) ([]string, error) {
	return nil, nil
}
