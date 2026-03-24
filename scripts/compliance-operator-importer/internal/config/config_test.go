package config

import (
	"testing"
	"time"

	"github.com/stackrox/co-acs-importer/internal/models"
)

// minimalValidArgs returns a set of args that always satisfies all required
// flags when token env var is pre-set by the caller.
func minimalValidArgs(overrides ...string) []string {
	base := []string{
		"--acs-endpoint", "https://central.example.com",
		"--co-namespace", "openshift-compliance",
		"--acs-cluster-id", "cluster-abc",
	}
	return append(base, overrides...)
}

// setenv is a test helper that sets an env var and returns a cleanup func.
func setenv(t *testing.T, key, value string) {
	t.Helper()
	t.Setenv(key, value)
}

// TestIMP_CLI_001_EndpointRequired verifies that omitting --acs-endpoint
// (with no ACS_ENDPOINT env var) produces an error.
func TestIMP_CLI_001_EndpointRequired(t *testing.T) {
	setenv(t, defaultTokenEnv, "tok")

	_, err := ParseAndValidate([]string{
		"--co-namespace", "openshift-compliance",
		"--acs-cluster-id", "cluster-abc",
	})
	if err == nil {
		t.Fatal("expected error for missing --acs-endpoint, got nil")
	}
}

// TestIMP_CLI_001_EndpointFromEnv verifies that ACS_ENDPOINT env var is
// accepted in place of --acs-endpoint.
func TestIMP_CLI_001_EndpointFromEnv(t *testing.T) {
	setenv(t, "ACS_ENDPOINT", "https://central.example.com")
	setenv(t, defaultTokenEnv, "tok")

	cfg, err := ParseAndValidate([]string{
		"--co-namespace", "openshift-compliance",
		"--acs-cluster-id", "cluster-abc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ACSEndpoint != "https://central.example.com" {
		t.Errorf("expected endpoint from env, got %q", cfg.ACSEndpoint)
	}
}

// TestIMP_CLI_013_HTTPSEnforced verifies that non-https endpoints are rejected.
func TestIMP_CLI_013_HTTPSEnforced(t *testing.T) {
	setenv(t, defaultTokenEnv, "tok")

	cases := []string{
		"http://central.example.com",
		"central.example.com",
		"ftp://central.example.com",
	}
	for _, endpoint := range cases {
		t.Run(endpoint, func(t *testing.T) {
			_, err := ParseAndValidate([]string{
				"--acs-endpoint", endpoint,
				"--co-namespace", "openshift-compliance",
				"--acs-cluster-id", "cluster-abc",
			})
			if err == nil {
				t.Fatalf("expected error for non-https endpoint %q, got nil", endpoint)
			}
		})
	}
}

// TestIMP_CLI_023_AuthModeEnum verifies that invalid auth modes are rejected.
func TestIMP_CLI_023_AuthModeEnum(t *testing.T) {
	setenv(t, defaultTokenEnv, "tok")

	_, err := ParseAndValidate(minimalValidArgs("--acs-auth-mode", "oauth"))
	if err == nil {
		t.Fatal("expected error for invalid auth mode 'oauth', got nil")
	}
}

// TestIMP_CLI_023_AuthModeTokenAccepted verifies that "token" is accepted.
func TestIMP_CLI_023_AuthModeTokenAccepted(t *testing.T) {
	setenv(t, defaultTokenEnv, "tok")

	cfg, err := ParseAndValidate(minimalValidArgs("--acs-auth-mode", "token"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AuthMode != models.AuthModeToken {
		t.Errorf("expected token mode, got %q", cfg.AuthMode)
	}
}

// TestIMP_CLI_023_AuthModeBasicAccepted verifies that "basic" is accepted.
func TestIMP_CLI_023_AuthModeBasicAccepted(t *testing.T) {
	setenv(t, defaultPasswordEnv, "secret")

	cfg, err := ParseAndValidate(minimalValidArgs(
		"--acs-auth-mode", "basic",
		"--acs-username", "admin",
	))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AuthMode != models.AuthModeBasic {
		t.Errorf("expected basic mode, got %q", cfg.AuthMode)
	}
}

// TestIMP_CLI_024_BasicModeFields verifies that basic mode reads username and
// password from the expected sources.
func TestIMP_CLI_024_BasicModeFields(t *testing.T) {
	setenv(t, "ACS_PASSWORD", "s3cr3t")
	setenv(t, "ACS_USERNAME", "alice")

	cfg, err := ParseAndValidate(minimalValidArgs("--acs-auth-mode", "basic"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Username != "alice" {
		t.Errorf("expected username alice, got %q", cfg.Username)
	}
	if cfg.PasswordEnv != defaultPasswordEnv {
		t.Errorf("expected password env %q, got %q", defaultPasswordEnv, cfg.PasswordEnv)
	}
}

// TestIMP_CLI_025_AmbiguousAuthMissingPassword verifies that basic mode without
// a password is rejected.
func TestIMP_CLI_025_AmbiguousAuthMissingPassword(t *testing.T) {
	// Ensure the password env var is absent.
	t.Setenv("ACS_PASSWORD", "")

	_, err := ParseAndValidate(minimalValidArgs(
		"--acs-auth-mode", "basic",
		"--acs-username", "admin",
	))
	if err == nil {
		t.Fatal("expected error for basic mode without password, got nil")
	}
}

// TestIMP_CLI_025_AmbiguousAuthMissingUsername verifies that basic mode without
// a username is rejected.
func TestIMP_CLI_025_AmbiguousAuthMissingUsername(t *testing.T) {
	setenv(t, "ACS_PASSWORD", "secret")
	// Ensure username env is absent.
	t.Setenv("ACS_USERNAME", "")

	_, err := ParseAndValidate(minimalValidArgs(
		"--acs-auth-mode", "basic",
		// No --acs-username
	))
	if err == nil {
		t.Fatal("expected error for basic mode without username, got nil")
	}
}

// TestIMP_CLI_025_AmbiguousAuthMissingToken verifies that token mode without a
// token is rejected.
func TestIMP_CLI_025_AmbiguousAuthMissingToken(t *testing.T) {
	t.Setenv(defaultTokenEnv, "")

	_, err := ParseAndValidate(minimalValidArgs("--acs-auth-mode", "token"))
	if err == nil {
		t.Fatal("expected error for token mode without token, got nil")
	}
}

// TestIMP_CLI_026_DefaultAuthModeIsToken verifies that when --acs-auth-mode is
// not set, the importer defaults to token mode.
func TestIMP_CLI_026_DefaultAuthModeIsToken(t *testing.T) {
	setenv(t, defaultTokenEnv, "tok")

	cfg, err := ParseAndValidate(minimalValidArgs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AuthMode != models.AuthModeToken {
		t.Errorf("expected default auth mode to be %q, got %q", models.AuthModeToken, cfg.AuthMode)
	}
}

// TestDefaultTimeout verifies the default request timeout is 30s.
func TestDefaultTimeout(t *testing.T) {
	setenv(t, defaultTokenEnv, "tok")

	cfg, err := ParseAndValidate(minimalValidArgs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.RequestTimeout != 30*time.Second {
		t.Errorf("expected 30s timeout, got %v", cfg.RequestTimeout)
	}
}

// TestDefaultMaxRetries verifies the default max retries is 5.
func TestDefaultMaxRetries(t *testing.T) {
	setenv(t, defaultTokenEnv, "tok")

	cfg, err := ParseAndValidate(minimalValidArgs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MaxRetries != defaultMaxRetries {
		t.Errorf("expected max retries %d, got %d", defaultMaxRetries, cfg.MaxRetries)
	}
}

// TestMissingACSClusterID verifies that omitting --acs-cluster-id is an error.
func TestMissingACSClusterID(t *testing.T) {
	setenv(t, defaultTokenEnv, "tok")

	_, err := ParseAndValidate([]string{
		"--acs-endpoint", "https://central.example.com",
		"--co-namespace", "openshift-compliance",
		// No --acs-cluster-id
	})
	if err == nil {
		t.Fatal("expected error for missing --acs-cluster-id, got nil")
	}
}

// TestMissingNamespaceScope verifies that providing neither --co-namespace nor
// --co-all-namespaces is an error.
func TestMissingNamespaceScope(t *testing.T) {
	setenv(t, defaultTokenEnv, "tok")

	_, err := ParseAndValidate([]string{
		"--acs-endpoint", "https://central.example.com",
		"--acs-cluster-id", "cluster-abc",
	})
	if err == nil {
		t.Fatal("expected error for missing namespace scope, got nil")
	}
}

// TestMutuallyExclusiveNamespaceFlags verifies that --co-namespace and
// --co-all-namespaces are mutually exclusive.
func TestMutuallyExclusiveNamespaceFlags(t *testing.T) {
	setenv(t, defaultTokenEnv, "tok")

	_, err := ParseAndValidate(minimalValidArgs("--co-all-namespaces"))
	if err == nil {
		t.Fatal("expected error for both --co-namespace and --co-all-namespaces, got nil")
	}
}

// TestAllNamespacesFlag verifies that --co-all-namespaces works without --co-namespace.
func TestAllNamespacesFlag(t *testing.T) {
	setenv(t, defaultTokenEnv, "tok")

	cfg, err := ParseAndValidate([]string{
		"--acs-endpoint", "https://central.example.com",
		"--acs-cluster-id", "cluster-abc",
		"--co-all-namespaces",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.COAllNamespaces {
		t.Error("expected COAllNamespaces=true")
	}
	if cfg.CONamespace != "" {
		t.Errorf("expected empty CONamespace, got %q", cfg.CONamespace)
	}
}

// TestNegativeMaxRetriesRejected verifies that --max-retries < 0 is rejected.
func TestNegativeMaxRetriesRejected(t *testing.T) {
	setenv(t, defaultTokenEnv, "tok")

	_, err := ParseAndValidate(minimalValidArgs("--max-retries", "-1"))
	if err == nil {
		t.Fatal("expected error for negative max-retries, got nil")
	}
}

// TestTrailingSlashStripped verifies that a trailing slash on the endpoint is
// stripped for consistency.
func TestTrailingSlashStripped(t *testing.T) {
	setenv(t, defaultTokenEnv, "tok")

	cfg, err := ParseAndValidate(minimalValidArgs(
		"--acs-endpoint", "https://central.example.com/",
	))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ACSEndpoint != "https://central.example.com" {
		t.Errorf("expected trailing slash stripped, got %q", cfg.ACSEndpoint)
	}
}
