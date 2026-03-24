// Package preflight verifies that the ACS endpoint is reachable and the
// supplied credentials are accepted before any resource processing begins.
package preflight

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/stackrox/co-acs-importer/internal/models"
)

const preflightPath = "/v2/compliance/scan/configurations?pagination.limit=1"

// Run performs preflight checks in order:
//  1. Verify endpoint uses https:// (IMP-CLI-013).
//  2. Verify auth material is non-empty for the configured mode (IMP-CLI-014).
//  3. Probe GET /v2/compliance/scan/configurations?pagination.limit=1 (IMP-CLI-015).
//  4. HTTP 401/403 => fail-fast with a remediation message (IMP-CLI-016).
//
// Returns nil on success, or an error with a remediation hint on failure.
func Run(ctx context.Context, cfg *models.Config) error {
	// IMP-CLI-013: endpoint must be https://.
	if !strings.HasPrefix(cfg.ACSEndpoint, "https://") {
		return fmt.Errorf(
			"preflight failed: endpoint %q must start with https:// (IMP-CLI-013)\n"+
				"Fix: use --acs-endpoint https://<host>",
			cfg.ACSEndpoint,
		)
	}

	// IMP-CLI-014: auth material must be non-empty.
	if err := checkAuthMaterial(cfg); err != nil {
		return err
	}

	client, err := buildHTTPClient(cfg)
	if err != nil {
		return fmt.Errorf("preflight failed: could not build HTTP client: %w", err)
	}

	url := cfg.ACSEndpoint + preflightPath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("preflight failed: could not build request: %w", err)
	}

	addAuthHeader(req, cfg)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf(
			"preflight failed: could not reach ACS at %s: %w\n"+
				"Fix: check network connectivity and that --acs-endpoint is correct",
			cfg.ACSEndpoint, err,
		)
	}
	defer resp.Body.Close()

	// IMP-CLI-015: success only on HTTP 200.
	// IMP-CLI-016: 401/403 => fail-fast with remediation message.
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		return errors.New(
			"preflight failed: ACS returned 401 Unauthorized (IMP-CLI-016)\n" +
				"Fix: verify your ACS API token or credentials are correct and not expired",
		)
	case http.StatusForbidden:
		return errors.New(
			"preflight failed: ACS returned 403 Forbidden (IMP-CLI-016)\n" +
				"Fix: ensure your ACS user has the 'Compliance' permission set with at least read access",
		)
	default:
		return fmt.Errorf(
			"preflight failed: ACS returned unexpected status %d from %s\n"+
				"Fix: verify the ACS endpoint is correct and the service is healthy",
			resp.StatusCode, url,
		)
	}
}

// checkAuthMaterial validates that the auth credentials for the configured
// mode are non-empty (IMP-CLI-014).
func checkAuthMaterial(cfg *models.Config) error {
	switch cfg.AuthMode {
	case models.AuthModeToken:
		token := os.Getenv(cfg.TokenEnv)
		if token == "" {
			return fmt.Errorf(
				"preflight failed: token auth mode requires a non-empty token in env var %q (IMP-CLI-014)\n"+
					"Fix: set %s=<your-api-token>",
				cfg.TokenEnv, cfg.TokenEnv,
			)
		}
	case models.AuthModeBasic:
		if cfg.Username == "" {
			return errors.New(
				"preflight failed: basic auth mode requires a non-empty username (IMP-CLI-014)\n" +
					"Fix: pass --acs-username=<user> or set ACS_USERNAME=<user>",
			)
		}
		password := os.Getenv(cfg.PasswordEnv)
		if password == "" {
			return fmt.Errorf(
				"preflight failed: basic auth mode requires a non-empty password in env var %q (IMP-CLI-014)\n"+
					"Fix: set %s=<your-password>",
				cfg.PasswordEnv, cfg.PasswordEnv,
			)
		}
	}
	return nil
}

// buildHTTPClient constructs an HTTP client with the TLS settings from cfg.
func buildHTTPClient(cfg *models.Config) (*http.Client, error) {
	tlsCfg := &tls.Config{
		InsecureSkipVerify: cfg.InsecureSkipVerify, //nolint:gosec // controlled by explicit flag
	}

	if cfg.CACertFile != "" {
		pem, err := os.ReadFile(cfg.CACertFile)
		if err != nil {
			return nil, fmt.Errorf("reading CA cert file %q: %w", cfg.CACertFile, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("CA cert file %q contains no valid PEM certificates", cfg.CACertFile)
		}
		tlsCfg.RootCAs = pool
	}

	transport := &http.Transport{TLSClientConfig: tlsCfg}
	return &http.Client{
		Transport: transport,
		Timeout:   cfg.RequestTimeout,
	}, nil
}

// addAuthHeader sets the Authorization header on req according to cfg.AuthMode.
func addAuthHeader(req *http.Request, cfg *models.Config) {
	switch cfg.AuthMode {
	case models.AuthModeToken:
		token := os.Getenv(cfg.TokenEnv)
		req.Header.Set("Authorization", "Bearer "+token)
	case models.AuthModeBasic:
		password := os.Getenv(cfg.PasswordEnv)
		creds := base64.StdEncoding.EncodeToString([]byte(cfg.Username + ":" + password))
		req.Header.Set("Authorization", "Basic "+creds)
	}
}
