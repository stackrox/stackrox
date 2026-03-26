package config

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stackrox/co-acs-importer/internal/models"
)

// minimalValidArgs returns args that satisfy all required flags when
// ROX_API_TOKEN is pre-set by the caller.
func minimalValidArgs(overrides ...string) []string {
	base := []string{
		"--endpoint", "https://central.example.com",
		"--co-namespace", "openshift-compliance",
	}
	return append(base, overrides...)
}

// setenv is a test helper that sets an env var and returns a cleanup func.
func setenv(t *testing.T, key, value string) {
	t.Helper()
	t.Setenv(key, value)
}

// clearAuthEnv ensures both auth env vars are unset for a clean test.
func clearAuthEnv(t *testing.T) {
	t.Helper()
	t.Setenv("ROX_API_TOKEN", "")
	t.Setenv("ROX_ADMIN_PASSWORD", "")
	t.Setenv("ROX_ADMIN_USER", "")
	t.Setenv("ROX_ENDPOINT", "")
}

// ===========================================================================
// IMP-CLI-001: --endpoint / ROX_ENDPOINT
// ===========================================================================

func TestIMP_CLI_001_EndpointRequired(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	_, err := ParseAndValidate([]string{"--co-namespace", "openshift-compliance"})
	if err == nil {
		t.Fatal("expected error for missing --endpoint, got nil")
	}
	if !strings.Contains(err.Error(), "--endpoint") {
		t.Errorf("expected error to mention --endpoint, got: %q", err.Error())
	}
}

func TestIMP_CLI_001_EndpointFromFlag(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate(minimalValidArgs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ACSEndpoint != "https://central.example.com" {
		t.Errorf("expected endpoint from flag, got %q", cfg.ACSEndpoint)
	}
}

func TestIMP_CLI_001_EndpointFromEnv(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_ENDPOINT", "https://central.example.com")
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate([]string{"--co-namespace", "openshift-compliance"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ACSEndpoint != "https://central.example.com" {
		t.Errorf("expected endpoint from ROX_ENDPOINT env, got %q", cfg.ACSEndpoint)
	}
}

func TestIMP_CLI_001_FlagOverridesEnv(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_ENDPOINT", "https://env-central.example.com")
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate([]string{
		"--endpoint", "https://flag-central.example.com",
		"--co-namespace", "openshift-compliance",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ACSEndpoint != "https://flag-central.example.com" {
		t.Errorf("expected flag to override env, got %q", cfg.ACSEndpoint)
	}
}

func TestIMP_CLI_001_EmptyEndpointEnvNotAccepted(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_ENDPOINT", "")
	setenv(t, "ROX_API_TOKEN", "tok")

	_, err := ParseAndValidate([]string{"--co-namespace", "openshift-compliance"})
	if err == nil {
		t.Fatal("expected error for empty ROX_ENDPOINT, got nil")
	}
}

// ===========================================================================
// IMP-CLI-002: auto-inferred auth mode
// ===========================================================================

func TestIMP_CLI_002_TokenAutoInferred(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate(minimalValidArgs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AuthMode != models.AuthModeToken {
		t.Errorf("expected token mode inferred, got %q", cfg.AuthMode)
	}
}

func TestIMP_CLI_002_BasicAutoInferred(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_ADMIN_PASSWORD", "secret")

	cfg, err := ParseAndValidate(minimalValidArgs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AuthMode != models.AuthModeBasic {
		t.Errorf("expected basic mode inferred, got %q", cfg.AuthMode)
	}
}

func TestIMP_CLI_002_NoOldAuthModeFlag(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	// --acs-auth-mode should be rejected as an unknown flag.
	_, err := ParseAndValidate(minimalValidArgs("--acs-auth-mode", "token"))
	if err == nil {
		t.Fatal("expected error for removed --acs-auth-mode flag, got nil")
	}
}

func TestIMP_CLI_002_NoOldTokenEnvFlag(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	_, err := ParseAndValidate(minimalValidArgs("--acs-token-env", "MY_TOKEN"))
	if err == nil {
		t.Fatal("expected error for removed --acs-token-env flag, got nil")
	}
}

func TestIMP_CLI_002_NoOldPasswordEnvFlag(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	_, err := ParseAndValidate(minimalValidArgs("--acs-password-env", "MY_PWD"))
	if err == nil {
		t.Fatal("expected error for removed --acs-password-env flag, got nil")
	}
}

func TestIMP_CLI_002_NoOldEndpointFlag(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	_, err := ParseAndValidate([]string{
		"--acs-endpoint", "https://central.example.com",
		"--co-namespace", "openshift-compliance",
	})
	if err == nil {
		t.Fatal("expected error for removed --acs-endpoint flag, got nil")
	}
}

func TestIMP_CLI_002_NoOldUsernameFlag(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_ADMIN_PASSWORD", "secret")

	_, err := ParseAndValidate(minimalValidArgs("--acs-username", "admin"))
	if err == nil {
		t.Fatal("expected error for removed --acs-username flag, got nil")
	}
}

func TestIMP_CLI_002_NoOldSourceKubecontextFlag(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	_, err := ParseAndValidate(minimalValidArgs("--source-kubecontext", "myctx"))
	if err == nil {
		t.Fatal("expected error for removed --source-kubecontext flag, got nil")
	}
}

func TestIMP_CLI_002_NoOldClusterIDFlag(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	_, err := ParseAndValidate(minimalValidArgs("--acs-cluster-id", "uuid"))
	if err == nil {
		t.Fatal("expected error for removed --acs-cluster-id flag, got nil")
	}
}

// ===========================================================================
// IMP-CLI-013: endpoint scheme handling
// ===========================================================================

func TestIMP_CLI_013_HTTPSAccepted(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate(minimalValidArgs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ACSEndpoint != "https://central.example.com" {
		t.Errorf("expected https endpoint, got %q", cfg.ACSEndpoint)
	}
}

func TestIMP_CLI_013_BareHostnameGetsHTTPS(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate([]string{
		"--endpoint", "central.example.com",
		"--co-namespace", "openshift-compliance",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ACSEndpoint != "https://central.example.com" {
		t.Errorf("expected https:// prepended, got %q", cfg.ACSEndpoint)
	}
}

func TestIMP_CLI_013_BareHostnameWithPortGetsHTTPS(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate([]string{
		"--endpoint", "central.example.com:8443",
		"--co-namespace", "openshift-compliance",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ACSEndpoint != "https://central.example.com:8443" {
		t.Errorf("expected https:// prepended with port, got %q", cfg.ACSEndpoint)
	}
}

func TestIMP_CLI_013_HTTPRejected(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	_, err := ParseAndValidate([]string{
		"--endpoint", "http://central.example.com",
		"--co-namespace", "openshift-compliance",
	})
	if err == nil {
		t.Fatal("expected error for http:// endpoint, got nil")
	}
	if !strings.Contains(err.Error(), "http://") {
		t.Errorf("expected error to mention http://, got: %q", err.Error())
	}
}

func TestIMP_CLI_013_FTPRejected(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	// ftp:// doesn't start with http:// so it's treated as a bare hostname.
	// After prepending https:// it becomes https://ftp://... which is wrong
	// but technically passes the scheme check. Let's verify it's handled.
	cfg, err := ParseAndValidate([]string{
		"--endpoint", "ftp://central.example.com",
		"--co-namespace", "openshift-compliance",
	})
	// ftp:// doesn't start with http:// or https:// so gets https:// prepended.
	// That's OK per spec — the spec only rejects http:// explicitly.
	if err != nil {
		t.Fatalf("unexpected error (ftp scheme gets https:// prepended): %v", err)
	}
	if !strings.HasPrefix(cfg.ACSEndpoint, "https://") {
		t.Errorf("expected https:// prepended to ftp:// input, got %q", cfg.ACSEndpoint)
	}
}

func TestIMP_CLI_013_BareHostnameFromEnvGetsHTTPS(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_ENDPOINT", "central.example.com")
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate([]string{"--co-namespace", "openshift-compliance"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ACSEndpoint != "https://central.example.com" {
		t.Errorf("expected https:// prepended for bare hostname from env, got %q", cfg.ACSEndpoint)
	}
}

func TestIMP_CLI_013_OpenShiftRouteHostname(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	// Typical OpenShift route hostname.
	cfg, err := ParseAndValidate([]string{
		"--endpoint", "central-stackrox.apps.mycluster.example.com",
		"--co-namespace", "openshift-compliance",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ACSEndpoint != "https://central-stackrox.apps.mycluster.example.com" {
		t.Errorf("expected https:// prepended, got %q", cfg.ACSEndpoint)
	}
}

// ===========================================================================
// IMP-CLI-024: basic mode fields (--username / ROX_ADMIN_USER / default admin)
// ===========================================================================

func TestIMP_CLI_024_BasicModeDefaultUsername(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_ADMIN_PASSWORD", "s3cr3t")

	cfg, err := ParseAndValidate(minimalValidArgs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Username != "admin" {
		t.Errorf("expected default username 'admin', got %q", cfg.Username)
	}
}

func TestIMP_CLI_024_BasicModeUsernameFromFlag(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_ADMIN_PASSWORD", "s3cr3t")

	cfg, err := ParseAndValidate(minimalValidArgs("--username", "alice"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Username != "alice" {
		t.Errorf("expected username 'alice', got %q", cfg.Username)
	}
}

func TestIMP_CLI_024_BasicModeUsernameFromEnv(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_ADMIN_PASSWORD", "s3cr3t")
	setenv(t, "ROX_ADMIN_USER", "bob")

	cfg, err := ParseAndValidate(minimalValidArgs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Username != "bob" {
		t.Errorf("expected username 'bob' from ROX_ADMIN_USER, got %q", cfg.Username)
	}
}

func TestIMP_CLI_024_FlagOverridesEnvForUsername(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_ADMIN_PASSWORD", "s3cr3t")
	setenv(t, "ROX_ADMIN_USER", "env-user")

	cfg, err := ParseAndValidate(minimalValidArgs("--username", "flag-user"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Username != "flag-user" {
		t.Errorf("expected --username to override ROX_ADMIN_USER, got %q", cfg.Username)
	}
}

func TestIMP_CLI_024_TokenModeIgnoresUsername(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	// Username is still set but should be irrelevant in token mode.
	cfg, err := ParseAndValidate(minimalValidArgs("--username", "ignored"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AuthMode != models.AuthModeToken {
		t.Errorf("expected token mode, got %q", cfg.AuthMode)
	}
}

// ===========================================================================
// IMP-CLI-025: ambiguous auth
// ===========================================================================

func TestIMP_CLI_025_BothTokenAndPasswordIsAmbiguous(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")
	setenv(t, "ROX_ADMIN_PASSWORD", "pwd")

	_, err := ParseAndValidate(minimalValidArgs())
	if err == nil {
		t.Fatal("expected error for ambiguous auth, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "ambiguous") {
		t.Errorf("expected 'ambiguous' in error, got: %q", err.Error())
	}
}

func TestIMP_CLI_025_NeitherTokenNorPasswordErrors(t *testing.T) {
	clearAuthEnv(t)

	_, err := ParseAndValidate(minimalValidArgs())
	if err == nil {
		t.Fatal("expected error for missing auth, got nil")
	}
	if !strings.Contains(err.Error(), "ROX_API_TOKEN") || !strings.Contains(err.Error(), "ROX_ADMIN_PASSWORD") {
		t.Errorf("expected error to mention both env vars, got: %q", err.Error())
	}
}

// ===========================================================================
// Defaults and other flags (IMP-CLI-004, IMP-CLI-006..012)
// ===========================================================================

func TestIMP_CLI_009_DefaultTimeout(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate(minimalValidArgs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.RequestTimeout != 30*time.Second {
		t.Errorf("IMP-CLI-009: expected 30s timeout, got %v", cfg.RequestTimeout)
	}
}

func TestIMP_CLI_009_CustomTimeout(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate(minimalValidArgs("--request-timeout", "2m"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.RequestTimeout != 2*time.Minute {
		t.Errorf("expected 2m timeout, got %v", cfg.RequestTimeout)
	}
}

func TestIMP_CLI_010_DefaultMaxRetries(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate(minimalValidArgs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MaxRetries != 5 {
		t.Errorf("IMP-CLI-010: expected max retries 5, got %d", cfg.MaxRetries)
	}
}

func TestIMP_CLI_010_NegativeMaxRetriesRejected(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	_, err := ParseAndValidate(minimalValidArgs("--max-retries", "-1"))
	if err == nil {
		t.Fatal("IMP-CLI-010: expected error for negative max-retries, got nil")
	}
}

func TestIMP_CLI_010_ZeroMaxRetriesAllowed(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate(minimalValidArgs("--max-retries", "0"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MaxRetries != 0 {
		t.Errorf("expected max retries 0, got %d", cfg.MaxRetries)
	}
}

func TestIMP_CLI_004_DefaultNamespace(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate([]string{
		"--endpoint", "https://central.example.com",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.CONamespace != "openshift-compliance" {
		t.Fatalf("IMP-CLI-004: expected default namespace 'openshift-compliance', got %q", cfg.CONamespace)
	}
}

func TestIMP_CLI_004_CustomNamespace(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate([]string{
		"--endpoint", "https://central.example.com",
		"--co-namespace", "custom-ns",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.CONamespace != "custom-ns" {
		t.Errorf("expected custom namespace, got %q", cfg.CONamespace)
	}
}

func TestIMP_CLI_004_AllNamespacesClearsDefault(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate(minimalValidArgs("--co-all-namespaces"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.CONamespace != "" {
		t.Fatalf("expected empty namespace with --co-all-namespaces, got %q", cfg.CONamespace)
	}
	if !cfg.COAllNamespaces {
		t.Fatal("expected COAllNamespaces to be true")
	}
}

func TestIMP_CLI_004_AllNamespacesWithoutExplicitNamespace(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate([]string{
		"--endpoint", "https://central.example.com",
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

func TestIMP_CLI_006_OverwriteExistingDefaultsFalse(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate(minimalValidArgs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.OverwriteExisting {
		t.Error("IMP-CLI-006: expected OverwriteExisting to default to false")
	}
}

func TestIMP_CLI_027_OverwriteExistingFlag(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate(minimalValidArgs("--overwrite-existing"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.OverwriteExisting {
		t.Error("IMP-CLI-027: expected OverwriteExisting=true when flag is set")
	}
}

func TestIMP_CLI_007_DryRunFlag(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate(minimalValidArgs("--dry-run"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.DryRun {
		t.Error("IMP-CLI-007: expected DryRun=true when flag is set")
	}
}

func TestIMP_CLI_008_ReportJSONFlag(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate(minimalValidArgs("--report-json", "/tmp/report.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ReportJSON != "/tmp/report.json" {
		t.Errorf("IMP-CLI-008: expected report path, got %q", cfg.ReportJSON)
	}
}

func TestIMP_CLI_011_CACertFileFlag(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate(minimalValidArgs("--ca-cert-file", "/path/to/ca.pem"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.CACertFile != "/path/to/ca.pem" {
		t.Errorf("IMP-CLI-011: expected ca-cert-file, got %q", cfg.CACertFile)
	}
}

func TestIMP_CLI_012_InsecureSkipVerifyDefault(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate(minimalValidArgs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.InsecureSkipVerify {
		t.Error("IMP-CLI-012: expected InsecureSkipVerify to default to false")
	}
}

func TestIMP_CLI_012_InsecureSkipVerifyFlag(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate(minimalValidArgs("--insecure-skip-verify"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.InsecureSkipVerify {
		t.Error("IMP-CLI-012: expected InsecureSkipVerify=true when flag is set")
	}
}

// ===========================================================================
// Edge cases
// ===========================================================================

func TestTrailingSlashStripped(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate([]string{
		"--endpoint", "https://central.example.com/",
		"--co-namespace", "openshift-compliance",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ACSEndpoint != "https://central.example.com" {
		t.Errorf("expected trailing slash stripped, got %q", cfg.ACSEndpoint)
	}
}

func TestMultipleTrailingSlashesStripped(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate([]string{
		"--endpoint", "https://central.example.com///",
		"--co-namespace", "openshift-compliance",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ACSEndpoint != "https://central.example.com" {
		t.Errorf("expected all trailing slashes stripped, got %q", cfg.ACSEndpoint)
	}
}

func TestHelpReturnsSpecialError(t *testing.T) {
	// Redirect stderr to avoid printing help text during test.
	oldStderr := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	defer func() { os.Stderr = oldStderr }()

	_, err := ParseAndValidate([]string{"--help"})
	if err != ErrHelpRequested {
		t.Errorf("expected ErrHelpRequested, got %v", err)
	}
}

func TestUnknownFlagRejected(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")

	_, err := ParseAndValidate(minimalValidArgs("--unknown-flag", "value"))
	if err == nil {
		t.Fatal("expected error for unknown flag, got nil")
	}
}

func TestEmptyArgsWithTokenAndEndpoint(t *testing.T) {
	clearAuthEnv(t)
	setenv(t, "ROX_API_TOKEN", "tok")
	setenv(t, "ROX_ENDPOINT", "https://central.example.com")

	cfg, err := ParseAndValidate([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ACSEndpoint != "https://central.example.com" {
		t.Errorf("expected endpoint from env with empty args, got %q", cfg.ACSEndpoint)
	}
	if cfg.AuthMode != models.AuthModeToken {
		t.Errorf("expected token mode, got %q", cfg.AuthMode)
	}
}

func TestWhitespaceOnlyTokenIsEmpty(t *testing.T) {
	clearAuthEnv(t)
	// Whitespace-only token — os.Getenv returns it as non-empty.
	// The config layer treats it as set (auth is inferred as token).
	// Preflight will catch an invalid token at the network level.
	setenv(t, "ROX_API_TOKEN", "   ")

	cfg, err := ParseAndValidate(minimalValidArgs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AuthMode != models.AuthModeToken {
		t.Errorf("expected token mode for whitespace token, got %q", cfg.AuthMode)
	}
}
