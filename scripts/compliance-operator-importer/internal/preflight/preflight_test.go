package preflight

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stackrox/co-acs-importer/internal/models"
)

// minimalTokenConfig returns a Config wired to the given server URL in token mode.
// Caller must set ROX_API_TOKEN env var.
func minimalTokenConfig(serverURL string) *models.Config {
	return &models.Config{
		ACSEndpoint:    serverURL,
		AuthMode:       models.AuthModeToken,
		RequestTimeout: 5 * time.Second,
	}
}

// TestIMP_CLI_015_200ResponseNoError verifies that a 200 response from the
// preflight probe returns nil (IMP-CLI-015).
func TestIMP_CLI_015_200ResponseNoError(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/v2/compliance/scan/configurations") {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	t.Setenv("ROX_API_TOKEN", "validtoken")

	cfg := minimalTokenConfig(srv.URL)
	cfg.InsecureSkipVerify = true

	err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("expected nil error for 200 response, got: %v", err)
	}
}

// TestIMP_CLI_016_401ReturnsRemediationError verifies that a 401 response
// causes a fail-fast error with remediation text (IMP-CLI-016).
func TestIMP_CLI_016_401ReturnsRemediationError(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	t.Setenv("ROX_API_TOKEN", "badtoken")

	cfg := minimalTokenConfig(srv.URL)
	cfg.InsecureSkipVerify = true

	err := Run(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for 401 response, got nil")
	}
	msg := err.Error()
	if !strings.Contains(strings.ToLower(msg), "unauthorized") && !strings.Contains(strings.ToLower(msg), "401") {
		t.Errorf("expected 'unauthorized' or '401' in error message, got: %q", msg)
	}
	if !strings.Contains(strings.ToLower(msg), "fix:") {
		t.Errorf("expected remediation hint (Fix:) in error message, got: %q", msg)
	}
}

// TestIMP_CLI_016_403ReturnsRemediationError verifies that a 403 response
// causes a fail-fast error with remediation text (IMP-CLI-016).
func TestIMP_CLI_016_403ReturnsRemediationError(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	t.Setenv("ROX_API_TOKEN", "insufficienttoken")

	cfg := minimalTokenConfig(srv.URL)
	cfg.InsecureSkipVerify = true

	err := Run(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for 403 response, got nil")
	}
	msg := err.Error()
	if !strings.Contains(strings.ToLower(msg), "forbidden") && !strings.Contains(strings.ToLower(msg), "403") {
		t.Errorf("expected 'forbidden' or '403' in error message, got: %q", msg)
	}
	if !strings.Contains(strings.ToLower(msg), "fix:") {
		t.Errorf("expected remediation hint (Fix:) in error message, got: %q", msg)
	}
}

// TestIMP_CLI_013_NonHTTPSEndpointRejected verifies that a non-https endpoint
// is rejected before any network call is made (IMP-CLI-013).
func TestIMP_CLI_013_NonHTTPSEndpointRejected(t *testing.T) {
	t.Setenv("ROX_API_TOKEN", "tok")

	cfg := &models.Config{
		ACSEndpoint:    "http://central.example.com",
		AuthMode:       models.AuthModeToken,
		RequestTimeout: 5 * time.Second,
	}

	err := Run(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for non-https endpoint, got nil")
	}
	if !strings.Contains(err.Error(), "https://") {
		t.Errorf("expected error to mention https://, got: %q", err.Error())
	}
}

// TestIMP_CLI_014_EmptyTokenRejected verifies that an empty token in token
// mode is caught before any HTTP request (IMP-CLI-014).
func TestIMP_CLI_014_EmptyTokenRejected(t *testing.T) {
	t.Setenv("ROX_API_TOKEN", "")

	cfg := &models.Config{
		ACSEndpoint:    "https://central.example.com",
		AuthMode:       models.AuthModeToken,
		RequestTimeout: 5 * time.Second,
	}

	err := Run(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for empty token, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "token") {
		t.Errorf("expected error message to mention token, got: %q", err.Error())
	}
}

// TestIMP_CLI_014_BasicModeEmptyPasswordRejected verifies that basic mode with
// an empty password is rejected before any HTTP request (IMP-CLI-014).
func TestIMP_CLI_014_BasicModeEmptyPasswordRejected(t *testing.T) {
	t.Setenv("ROX_ADMIN_PASSWORD", "")

	cfg := &models.Config{
		ACSEndpoint:    "https://central.example.com",
		AuthMode:       models.AuthModeBasic,
		Username:       "admin",
		RequestTimeout: 5 * time.Second,
	}

	err := Run(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for empty password in basic mode, got nil")
	}
}

// TestIMP_CLI_015_ProbesCorrectPath verifies that the preflight probe sends
// a request to the expected API path (IMP-CLI-015).
func TestIMP_CLI_015_ProbesCorrectPath(t *testing.T) {
	var capturedPath string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.RequestURI()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	t.Setenv("ROX_API_TOKEN", "tok")

	cfg := minimalTokenConfig(srv.URL)
	cfg.InsecureSkipVerify = true

	if err := Run(context.Background(), cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedPath := "/v2/compliance/scan/configurations?pagination.limit=1"
	if capturedPath != expectedPath {
		t.Errorf("expected probe path %q, got %q", expectedPath, capturedPath)
	}
}

// TestIMP_CLI_015_BearerTokenSentInHeader verifies that the Authorization
// header is set to "Bearer <token>" in token mode.
func TestIMP_CLI_015_BearerTokenSentInHeader(t *testing.T) {
	var capturedAuth string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	t.Setenv("ROX_API_TOKEN", "my-secret-token")

	cfg := minimalTokenConfig(srv.URL)
	cfg.InsecureSkipVerify = true

	if err := Run(context.Background(), cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedAuth != "Bearer my-secret-token" {
		t.Errorf("expected Authorization 'Bearer my-secret-token', got %q", capturedAuth)
	}
}
