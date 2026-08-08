package clairclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_CreateIndexReport(t *testing.T) {
	ctx := t.Context()
	expectedManifest := Manifest{
		Hash: "sha256:abc123",
		Layers: []Layer{
			{
				Hash: "sha256:layer1",
				URI:  "https://example.com/layer1",
				Headers: map[string][]string{
					"Authorization": {"Bearer token"},
				},
			},
		},
	}

	expectedReport := &IndexReport{
		ManifestHash: "sha256:abc123",
		State:        "IndexFinished",
		Success:      true,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/indexer/api/v1/index_report", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var receivedManifest Manifest
		err := json.NewDecoder(r.Body).Decode(&receivedManifest)
		require.NoError(t, err)
		assert.Equal(t, expectedManifest, receivedManifest)

		w.WriteHeader(http.StatusCreated)
		require.NoError(t, json.NewEncoder(w).Encode(expectedReport))
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	require.NoError(t, err)

	report, err := client.CreateIndexReport(ctx, expectedManifest)
	require.NoError(t, err)
	assert.Equal(t, expectedReport, report)
}

func TestClient_GetIndexReport(t *testing.T) {
	ctx := t.Context()
	digest := "sha256:abc123"

	expectedReport := &IndexReport{
		ManifestHash: digest,
		State:        "IndexFinished",
		Success:      true,
		Packages: map[string]Package{
			"1": {
				ID:      "1",
				Name:    "curl",
				Version: "7.68.0",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/indexer/api/v1/index_report/"+digest, r.URL.Path)

		w.WriteHeader(http.StatusOK)
		require.NoError(t, json.NewEncoder(w).Encode(expectedReport))
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	require.NoError(t, err)

	report, err := client.GetIndexReport(ctx, digest)
	require.NoError(t, err)
	assert.Equal(t, expectedReport, report)
}

func TestClient_GetIndexReport_NotFound(t *testing.T) {
	ctx := t.Context()
	digest := "sha256:nonexistent"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/indexer/api/v1/index_report/"+digest, r.URL.Path)

		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("index report not found"))
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	require.NoError(t, err)

	report, err := client.GetIndexReport(ctx, digest)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Nil(t, report)
}

func TestClient_GetVulnerabilityReport(t *testing.T) {
	ctx := t.Context()
	digest := "sha256:abc123"

	expectedReport := &VulnerabilityReport{
		ManifestHash: digest,
		State:        "Matched",
		Success:      true,
		Vulnerabilities: map[string]Vulnerability{
			"CVE-2021-1234": {
				ID:          "CVE-2021-1234",
				Name:        "CVE-2021-1234",
				Description: "A vulnerability",
				Severity:    "High",
				Issued:      time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		PackageVulnerabilities: map[string][]string{
			"1": {"CVE-2021-1234"},
		},
	}

	tests := map[string]struct {
		statusCode int
	}{
		"200 OK": {
			statusCode: http.StatusOK,
		},
		"201 Created": {
			statusCode: http.StatusCreated,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/matcher/api/v1/vulnerability_report/"+digest, r.URL.Path)

				w.WriteHeader(tc.statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(expectedReport))
			}))
			defer server.Close()

			client, err := NewClient(server.URL)
			require.NoError(t, err)

			report, err := client.GetVulnerabilityReport(ctx, digest)
			require.NoError(t, err)
			assert.Equal(t, expectedReport, report)
		})
	}
}

func TestClient_DeleteIndexReport(t *testing.T) {
	ctx := t.Context()
	digest := "sha256:abc123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/indexer/api/v1/index_report/"+digest, r.URL.Path)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	require.NoError(t, err)

	err = client.DeleteIndexReport(ctx, digest)
	require.NoError(t, err)
}

func TestClient_GetIndexState(t *testing.T) {
	ctx := t.Context()

	expectedState := &IndexState{
		State: "IndexFinished",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/indexer/api/v1/index_state", r.URL.Path)

		w.WriteHeader(http.StatusOK)
		require.NoError(t, json.NewEncoder(w).Encode(expectedState))
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	require.NoError(t, err)

	state, err := client.GetIndexState(ctx)
	require.NoError(t, err)
	assert.Equal(t, expectedState, state)
}

func TestClient_GetUpdateOperations(t *testing.T) {
	ctx := t.Context()

	expectedOps := map[string][]UpdateOperation{
		"alpine": {
			{
				Ref:     "v3.15",
				Updater: "alpine",
				Date:    time.Date(2021, 12, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		"debian": {
			{
				Ref:     "bullseye",
				Updater: "debian",
				Date:    time.Date(2021, 11, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/matcher/api/v1/internal/update_operation", r.URL.Path)
		assert.Equal(t, "true", r.URL.Query().Get("latest"))

		w.WriteHeader(http.StatusOK)
		require.NoError(t, json.NewEncoder(w).Encode(expectedOps))
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	require.NoError(t, err)

	ops, err := client.GetUpdateOperations(ctx)
	require.NoError(t, err)
	assert.Equal(t, expectedOps, ops)
}

func TestClient_WithHTTPClient(t *testing.T) {
	ctx := t.Context()

	customClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		require.NoError(t, json.NewEncoder(w).Encode(&IndexState{State: "test"}))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, WithHTTPClient(customClient))
	require.NoError(t, err)
	assert.Same(t, customClient, client.httpClient)

	_, err = client.GetIndexState(ctx)
	require.NoError(t, err)
}

func TestClient_ErrorHandling(t *testing.T) {
	ctx := t.Context()

	tests := map[string]struct {
		statusCode   int
		responseBody string
		expectedErr  string
	}{
		"400 Bad Request": {
			statusCode:   http.StatusBadRequest,
			responseBody: "invalid manifest format",
			expectedErr:  "HTTP 400: invalid manifest format",
		},
		"500 Internal Server Error": {
			statusCode:   http.StatusInternalServerError,
			responseBody: "database connection failed",
			expectedErr:  "HTTP 500: database connection failed",
		},
		"503 Service Unavailable": {
			statusCode:   http.StatusServiceUnavailable,
			responseBody: "service temporarily unavailable",
			expectedErr:  "HTTP 503: service temporarily unavailable",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte(tc.responseBody))
			}))
			defer server.Close()

			client, err := NewClient(server.URL)
			require.NoError(t, err)

			manifest := Manifest{Hash: "sha256:test"}
			_, err = client.CreateIndexReport(ctx, manifest)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedErr)
		})
	}
}

func TestClient_LargeErrorBody(t *testing.T) {
	ctx := t.Context()

	// Create error body larger than 4096 bytes
	largeError := make([]byte, 5000)
	for i := range largeError {
		largeError[i] = 'x'
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(largeError)
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	require.NoError(t, err)

	_, err = client.GetIndexState(ctx)
	require.Error(t, err)
	// Error message should be truncated to 4096 bytes
	assert.LessOrEqual(t, len(err.Error()), 4096+50) // Account for "HTTP 500: " prefix
}

func TestClient_NewClient_InvalidURL(t *testing.T) {
	_, err := NewClient("://invalid-url")
	require.Error(t, err)
}

func TestClient_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay response to allow context cancellation
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	require.NoError(t, err)

	// Cancel context before request completes
	cancel()

	_, err = client.GetIndexState(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}
