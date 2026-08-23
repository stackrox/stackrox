package matcher

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stackrox/rox/clair-adapter/clairclient"
	"github.com/stackrox/rox/clair-adapter/enricher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatcher_GetVulnerabilities(t *testing.T) {
	mockVuln := clairclient.Vulnerability{
		ID:             "CVE-2024-1234",
		Name:           "Test Vulnerability",
		Severity:       "High",
		FixedInVersion: "1.2.3",
		Package: &clairclient.Package{
			ID:      "pkg1",
			Name:    "testpkg",
			Version: "1.0.0",
		},
	}

	mockReport := &clairclient.VulnerabilityReport{
		ManifestHash: "sha256:test123",
		State:        "MatchFinished",
		Success:      true,
		Packages: map[string]clairclient.Package{
			"pkg1": {
				ID:      "pkg1",
				Name:    "testpkg",
				Version: "1.0.0",
			},
		},
		Vulnerabilities: map[string]clairclient.Vulnerability{
			"CVE-2024-1234": mockVuln,
		},
		PackageVulnerabilities: map[string][]string{
			"pkg1": {"CVE-2024-1234"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Contains(t, r.URL.Path, "vulnerability_report")

		w.WriteHeader(http.StatusCreated)
		err := json.NewEncoder(w).Encode(mockReport)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := clairclient.NewClient(server.URL)
	require.NoError(t, err)

	pipeline := enricher.NewPipeline()
	matcher := NewLocalMatcher(client, pipeline, nil)

	ctx := context.Background()
	report, enrichmentResult, err := matcher.GetVulnerabilities(ctx, "sha256:test123")

	require.NoError(t, err)
	assert.NotNil(t, report)
	assert.NotNil(t, enrichmentResult)

	// Verify report content
	assert.Equal(t, "sha256:test123", report.ManifestHash)
	assert.True(t, report.Success)
	assert.Contains(t, report.Vulnerabilities, "CVE-2024-1234")

	// Verify enrichment result has PkgFixedBy populated
	assert.NotNil(t, enrichmentResult.PkgFixedBy)
	assert.Contains(t, enrichmentResult.PkgFixedBy, "pkg1")
	assert.Equal(t, "1.2.3", enrichmentResult.PkgFixedBy["pkg1"])
}

func TestMatcher_GetVulnerabilities_MultipleVulns(t *testing.T) {
	mockReport := &clairclient.VulnerabilityReport{
		ManifestHash: "sha256:multi",
		State:        "MatchFinished",
		Success:      true,
		Packages: map[string]clairclient.Package{
			"pkg1": {ID: "pkg1", Name: "pkg1", Version: "1.0"},
			"pkg2": {ID: "pkg2", Name: "pkg2", Version: "2.0"},
		},
		Vulnerabilities: map[string]clairclient.Vulnerability{
			"CVE-2024-0001": {
				ID:             "CVE-2024-0001",
				FixedInVersion: "1.1.0",
				Package:        &clairclient.Package{ID: "pkg1"},
			},
			"CVE-2024-0002": {
				ID:             "CVE-2024-0002",
				FixedInVersion: "2.1.0",
				Package:        &clairclient.Package{ID: "pkg2"},
			},
		},
		PackageVulnerabilities: map[string][]string{
			"pkg1": {"CVE-2024-0001"},
			"pkg2": {"CVE-2024-0002"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(mockReport)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := clairclient.NewClient(server.URL)
	require.NoError(t, err)

	pipeline := enricher.NewPipeline()
	matcher := NewLocalMatcher(client, pipeline, nil)

	ctx := context.Background()
	_, enrichmentResult, err := matcher.GetVulnerabilities(ctx, "sha256:multi")

	require.NoError(t, err)
	assert.NotNil(t, enrichmentResult)
	assert.Len(t, enrichmentResult.PkgFixedBy, 2)
	assert.Equal(t, "1.1.0", enrichmentResult.PkgFixedBy["pkg1"])
	assert.Equal(t, "2.1.0", enrichmentResult.PkgFixedBy["pkg2"])
}

func TestMatcher_GetLastVulnerabilityUpdate(t *testing.T) {
	now := time.Now()
	oldDate := now.Add(-48 * time.Hour)
	recentDate := now.Add(-1 * time.Hour)

	mockOperations := map[string][]clairclient.UpdateOperation{
		"updater1": {
			{
				Ref:     "ref1",
				Updater: "updater1",
				Date:    oldDate,
			},
		},
		"updater2": {
			{
				Ref:     "ref2",
				Updater: "updater2",
				Date:    recentDate,
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Contains(t, r.URL.Path, "update_operation")
		assert.Equal(t, "true", r.URL.Query().Get("latest"))

		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(mockOperations)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := clairclient.NewClient(server.URL)
	require.NoError(t, err)

	pipeline := enricher.NewPipeline()
	matcher := NewLocalMatcher(client, pipeline, nil)

	ctx := context.Background()
	lastUpdate, err := matcher.GetLastVulnerabilityUpdate(ctx)

	require.NoError(t, err)
	// Should return the most recent date
	assert.True(t, lastUpdate.After(oldDate))
	assert.WithinDuration(t, recentDate, lastUpdate, time.Second)
}

func TestMatcher_GetLastVulnerabilityUpdate_WithMetadataStore(t *testing.T) {
	expectedTime := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

	mockStore := &mockMatcherMetadataStore{
		lastUpdate: expectedTime,
	}

	// Don't need a real server since we won't hit it
	client, err := clairclient.NewClient("http://localhost:9999")
	require.NoError(t, err)

	pipeline := enricher.NewPipeline()
	matcher := NewLocalMatcher(client, pipeline, mockStore)

	ctx := context.Background()
	lastUpdate, err := matcher.GetLastVulnerabilityUpdate(ctx)

	require.NoError(t, err)
	assert.Equal(t, expectedTime, lastUpdate)
}

func TestMatcher_GetLastVulnerabilityUpdate_EmptyOperations(t *testing.T) {
	mockOperations := map[string][]clairclient.UpdateOperation{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(mockOperations)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := clairclient.NewClient(server.URL)
	require.NoError(t, err)

	pipeline := enricher.NewPipeline()
	matcher := NewLocalMatcher(client, pipeline, nil)

	ctx := context.Background()
	lastUpdate, err := matcher.GetLastVulnerabilityUpdate(ctx)

	require.NoError(t, err)
	assert.True(t, lastUpdate.IsZero())
}

func TestMatcher_GetVulnerabilities_ErrorFromClair(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := clairclient.NewClient(server.URL)
	require.NoError(t, err)

	pipeline := enricher.NewPipeline()
	matcher := NewLocalMatcher(client, pipeline, nil)

	ctx := context.Background()
	report, enrichmentResult, err := matcher.GetVulnerabilities(ctx, "sha256:error")

	assert.Error(t, err)
	assert.Nil(t, report)
	assert.Nil(t, enrichmentResult)
}

// mockMatcherMetadataStore is a simple in-memory implementation for testing
type mockMatcherMetadataStore struct {
	lastUpdate time.Time
}

func (m *mockMatcherMetadataStore) GetLastVulnerabilityUpdate(ctx context.Context) (time.Time, error) {
	return m.lastUpdate, nil
}

func (m *mockMatcherMetadataStore) SetLastVulnerabilityUpdate(ctx context.Context, bundle string, lastUpdate time.Time) error {
	m.lastUpdate = lastUpdate
	return nil
}
