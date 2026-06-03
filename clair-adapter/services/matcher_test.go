package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stackrox/rox/clair-adapter/clairclient"
	"github.com/stackrox/rox/clair-adapter/enricher"
	"github.com/stackrox/rox/clair-adapter/matcher"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// newTestMatcherService creates a test MatcherService backed by a mock Clair server.
func newTestMatcherService(t *testing.T, handler http.Handler) *MatcherService {
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c, err := clairclient.NewClient(srv.URL)
	require.NoError(t, err)

	pipeline := enricher.NewPipeline()
	m := matcher.NewLocalMatcher(c, pipeline, nil)
	return NewMatcherService(m)
}

func TestGetVulnerabilities_Success(t *testing.T) {
	const hashID = "sha256:abc123"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/matcher/api/v1/vulnerability_report/"+hashID {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Return a vulnerability report with a vulnerability
		report := &clairclient.VulnerabilityReport{
			ManifestHash: hashID,
			Vulnerabilities: map[string]clairclient.Vulnerability{
				"vuln-1": {
					ID:                 "vuln-1",
					Name:               "CVE-2023-1234",
					Description:        "A test vulnerability",
					Severity:           "high",
					NormalizedSeverity: "high",
					FixedInVersion:     "1.2.3",
					Links:              "https://example.com/cve-2023-1234",
					Issued:             time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC),
				},
			},
			PackageVulnerabilities: map[string][]string{
				"pkg-1": {"vuln-1"},
			},
			Packages: map[string]clairclient.Package{
				"pkg-1": {
					ID:      "pkg-1",
					Name:    "test-package",
					Version: "1.0.0",
					Kind:    "binary",
				},
			},
			Enrichments: map[string][]json.RawMessage{},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(report)
	})

	svc := newTestMatcherService(t, handler)

	req := &v4.GetVulnerabilitiesRequest{
		HashId: hashID,
	}

	resp, err := svc.GetVulnerabilities(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, hashID, resp.GetHashId())
	assert.Len(t, resp.GetVulnerabilities(), 1)
	assert.Contains(t, resp.GetVulnerabilities(), "vuln-1")
}

func TestGetVulnerabilities_NotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"code":    "404",
			"message": "vulnerability report not found",
		})
	})

	svc := newTestMatcherService(t, handler)

	req := &v4.GetVulnerabilitiesRequest{
		HashId: "sha256:nonexistent",
	}

	resp, err := svc.GetVulnerabilities(context.Background(), req)
	require.Error(t, err)
	assert.Nil(t, resp)

	// Check for NotFound status
	st := status.Convert(err)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestGetMetadata_Success(t *testing.T) {
	lastUpdate := time.Date(2023, 6, 15, 12, 0, 0, 0, time.UTC)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/matcher/api/v1/internal/update_operation" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Return update operations
		updateOps := map[string][]clairclient.UpdateOperation{
			"updater-1": {
				{
					Updater: "updater-1",
					Date:    lastUpdate,
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(updateOps)
	})

	svc := newTestMatcherService(t, handler)

	resp, err := svc.GetMetadata(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Check timestamp matches
	assert.NotNil(t, resp.GetLastVulnerabilityUpdate())
	assert.Equal(t, lastUpdate.Unix(), resp.GetLastVulnerabilityUpdate().GetSeconds())
}
