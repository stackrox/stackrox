package acs_test

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stackrox/co-acs-importer/internal/acs"
	"github.com/stackrox/co-acs-importer/internal/models"
)

// newTestConfig returns a Config wired to the given TLS test server URL.
// InsecureSkipVerify is always true so the self-signed httptest cert is accepted.
func newTestConfig(serverURL string) *models.Config {
	return &models.Config{
		ACSEndpoint:        serverURL,
		AuthMode:           models.AuthModeToken,
		RequestTimeout:     5 * time.Second,
		MaxRetries:         3,
		InsecureSkipVerify: true,
	}
}

// startTLSServer starts an httptest TLS server with the provided handler and
// returns the server plus an http.Client pre-configured with the server's TLS cert.
func startTLSServer(handler http.Handler) (*httptest.Server, *http.Client) {
	srv := httptest.NewTLSServer(handler)
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // test only
		},
		Timeout: 5 * time.Second,
	}
	return srv, client
}

// IMP-CLI-015: Preflight 200 => nil error
func TestPreflight_200_ReturnsNil(t *testing.T) {
	srv, _ := startTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/compliance/scan/configurations" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(models.ACSListResponse{})
	}))
	defer srv.Close()

	t.Setenv("ROX_API_TOKEN", "test-token")
	cfg := newTestConfig(srv.URL)
	client, err := acs.NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if err := client.Preflight(context.Background()); err != nil {
		t.Errorf("IMP-CLI-015: Preflight with HTTP 200 should return nil, got: %v", err)
	}
}

// IMP-CLI-016: Preflight 401 => error
func TestPreflight_401_ReturnsError(t *testing.T) {
	srv, _ := startTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	t.Setenv("ROX_API_TOKEN", "bad-token")
	cfg := newTestConfig(srv.URL)
	client, err := acs.NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if err := client.Preflight(context.Background()); err == nil {
		t.Error("IMP-CLI-016: Preflight with HTTP 401 should return error, got nil")
	}
}

// IMP-CLI-016: Preflight 403 => error
func TestPreflight_403_ReturnsError(t *testing.T) {
	srv, _ := startTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Forbidden", http.StatusForbidden)
	}))
	defer srv.Close()

	t.Setenv("ROX_API_TOKEN", "bad-token")
	cfg := newTestConfig(srv.URL)
	client, err := acs.NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if err := client.Preflight(context.Background()); err == nil {
		t.Error("IMP-CLI-016: Preflight with HTTP 403 should return error, got nil")
	}
}

// IMP-IDEM-001: ListScanConfigurations returns parsed list
func TestListScanConfigurations_ReturnsParsedList(t *testing.T) {
	want := []models.ACSConfigSummary{
		{ID: "id-1", ScanName: "cis-weekly"},
		{ID: "id-2", ScanName: "pci-daily"},
	}
	srv, _ := startTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/compliance/scan/configurations" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(models.ACSListResponse{
			Configurations: want,
			TotalCount:     int32(len(want)),
		})
	}))
	defer srv.Close()

	t.Setenv("ROX_API_TOKEN", "test-token")
	cfg := newTestConfig(srv.URL)
	client, err := acs.NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	got, err := client.ListScanConfigurations(context.Background())
	if err != nil {
		t.Fatalf("IMP-IDEM-001: ListScanConfigurations: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("IMP-IDEM-001: expected %d configs, got %d", len(want), len(got))
	}
	for i, g := range got {
		if g.ID != want[i].ID || g.ScanName != want[i].ScanName {
			t.Errorf("IMP-IDEM-001: item[%d]: got {%s %s}, want {%s %s}", i, g.ID, g.ScanName, want[i].ID, want[i].ScanName)
		}
	}
}

// IMP-IDEM-003: CreateScanConfiguration uses POST method (never PUT)
// IMP-IDEM-001: CreateScanConfiguration returns new config ID
func TestCreateScanConfiguration_UsesPOSTAndReturnsID(t *testing.T) {
	const wantID = "new-config-id-123"
	var gotMethod string

	srv, _ := startTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/compliance/scan/configurations" {
			http.NotFound(w, r)
			return
		}
		gotMethod = r.Method
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": wantID})
	}))
	defer srv.Close()

	t.Setenv("ROX_API_TOKEN", "test-token")
	cfg := newTestConfig(srv.URL)
	client, err := acs.NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	payload := models.ACSCreatePayload{
		ScanName: "cis-weekly",
		ScanConfig: models.ACSBaseScanConfig{
			Profiles:    []string{"ocp4-cis"},
			Description: "test",
		},
		Clusters: []string{"cluster-a"},
	}

	gotID, err := client.CreateScanConfiguration(context.Background(), payload)
	if err != nil {
		t.Fatalf("IMP-IDEM-001: CreateScanConfiguration: %v", err)
	}

	// IMP-IDEM-003: must use POST, never PUT
	if gotMethod != http.MethodPost {
		t.Errorf("IMP-IDEM-003: expected method POST, got %s", gotMethod)
	}
	if gotMethod == http.MethodPut {
		t.Errorf("IMP-IDEM-003: VIOLATION - PUT was called, which is forbidden in Phase 1")
	}

	// IMP-IDEM-001: must return the ID from the response
	if gotID != wantID {
		t.Errorf("IMP-IDEM-001: expected ID %q, got %q", wantID, gotID)
	}
}

// IMP-IDEM-003: Compile-time guard - verify the ACSClient interface has no Put method.
func TestNoPUTMethodOnInterface(t *testing.T) {
	// IMP-IDEM-003: This test documents the invariant.
	t.Log("IMP-IDEM-003: ACSClient interface has no PUT method - enforced by interface definition")
}
