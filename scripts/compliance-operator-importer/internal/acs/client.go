// Package acs provides an HTTP client for the ACS compliance scan configuration API.
//
// create-only: PUT is never called in Phase 1
package acs

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/stackrox/co-acs-importer/internal/models"
)

// client is the concrete implementation of models.ACSClient.
// It issues only GET and POST requests. No PUT method exists in Phase 1.
type client struct {
	httpClient *http.Client
	baseURL    string
	cfg        *models.Config
}

// NewClient creates a models.ACSClient from cfg.
//
// TLS is configured from cfg.CACertFile and cfg.InsecureSkipVerify.
// Timeout is set from cfg.RequestTimeout.
// Authentication:
//   - token mode:  "Authorization: Bearer <token>" (token resolved from cfg.TokenEnv)
//   - basic mode: HTTP Basic auth (cfg.Username + password from cfg.PasswordEnv)
//
// create-only: PUT is never called in Phase 1
func NewClient(cfg *models.Config) (models.ACSClient, error) {
	tlsCfg, err := buildTLSConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("acs: building TLS config: %w", err)
	}

	transport := &http.Transport{
		TLSClientConfig: tlsCfg,
	}

	timeout := cfg.RequestTimeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   timeout,
		},
		baseURL: cfg.ACSEndpoint,
		cfg:     cfg,
	}, nil
}

// buildTLSConfig constructs a tls.Config from the importer config.
func buildTLSConfig(cfg *models.Config) (*tls.Config, error) {
	tlsCfg := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: cfg.InsecureSkipVerify, //nolint:gosec // controlled by explicit CLI flag
	}

	if cfg.CACertFile != "" {
		pemData, err := os.ReadFile(cfg.CACertFile)
		if err != nil {
			return nil, fmt.Errorf("reading CA cert file %q: %w", cfg.CACertFile, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pemData) {
			return nil, fmt.Errorf("no valid PEM certificates found in %q", cfg.CACertFile)
		}
		tlsCfg.RootCAs = pool
	}

	return tlsCfg, nil
}

// addAuth adds the correct Authorization header to req based on the configured auth mode.
func (c *client) addAuth(req *http.Request) error {
	switch c.cfg.AuthMode {
	case models.AuthModeBasic:
		password := os.Getenv(c.cfg.PasswordEnv)
		req.SetBasicAuth(c.cfg.Username, password)
	default: // token mode
		tokenEnv := c.cfg.TokenEnv
		if tokenEnv == "" {
			tokenEnv = "ACS_API_TOKEN"
		}
		token := os.Getenv(tokenEnv)
		if token == "" {
			return fmt.Errorf("acs: token env var %q is empty", tokenEnv)
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return nil
}

// Preflight checks ACS connectivity and auth by calling:
//
//	GET /v2/compliance/scan/configurations?pagination.limit=1
//
// Only HTTP 200 is treated as success; any other status returns an error.
//
// Implements IMP-CLI-015, IMP-CLI-016.
func (c *client) Preflight(ctx context.Context) error {
	url := c.baseURL + "/v2/compliance/scan/configurations?pagination.limit=1"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("acs: preflight request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if err := c.addAuth(req); err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("acs: preflight failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return errors.New("acs: preflight: HTTP 401 Unauthorized - check token or credentials")
		case http.StatusForbidden:
			return errors.New("acs: preflight: HTTP 403 Forbidden - token lacks required permissions")
		default:
			return fmt.Errorf("acs: preflight: unexpected HTTP %d", resp.StatusCode)
		}
	}
	return nil
}

// ListScanConfigurations returns all existing scan configuration summaries by calling:
//
//	GET /v2/compliance/scan/configurations?pagination.limit=1000
//
// Implements IMP-IDEM-001 (used to build the existing-name set).
func (c *client) ListScanConfigurations(ctx context.Context) ([]models.ACSConfigSummary, error) {
	url := c.baseURL + "/v2/compliance/scan/configurations?pagination.limit=1000"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("acs: list request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if err := c.addAuth(req); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("acs: list scan configurations: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("acs: list scan configurations: HTTP %d", resp.StatusCode)
	}

	var listResp models.ACSListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("acs: decoding list response: %w", err)
	}
	return listResp.Configurations, nil
}

// complianceScanConfigurationResponse is used to parse the id from the POST response.
type complianceScanConfigurationResponse struct {
	ID string `json:"id"`
}

// CreateScanConfiguration sends POST /v2/compliance/scan/configurations and returns
// the ID of the newly created configuration.
//
// IMPORTANT: This method MUST use POST only. No PUT is called anywhere in Phase 1.
// Implements IMP-IDEM-001, IMP-IDEM-003.
//
// create-only: PUT is never called in Phase 1
func (c *client) CreateScanConfiguration(ctx context.Context, payload models.ACSCreatePayload) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("acs: marshalling create payload: %w", err)
	}

	url := c.baseURL + "/v2/compliance/scan/configurations"
	// POST only - never PUT - create-only Phase 1
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("acs: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if err := c.addAuth(req); err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("acs: create scan configuration: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", &HTTPError{Code: resp.StatusCode, Message: fmt.Sprintf("POST /v2/compliance/scan/configurations returned HTTP %d", resp.StatusCode)}
	}

	var created complianceScanConfigurationResponse
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return "", fmt.Errorf("acs: decoding create response: %w", err)
	}
	if created.ID == "" {
		return "", errors.New("acs: create response contained empty id")
	}
	return created.ID, nil
}

// HTTPError is returned by CreateScanConfiguration when the server responds with
// a non-success HTTP status. The reconciler uses StatusCode() to decide whether
// to retry (transient: 429,502,503,504) or abort (non-transient: 400,401,403,404).
type HTTPError struct {
	Code    int
	Message string
}

func (e *HTTPError) Error() string   { return e.Message }
func (e *HTTPError) StatusCode() int { return e.Code }
