package clairclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// ErrNotFound is returned when a resource is not found (HTTP 404).
var ErrNotFound = errors.New("not found")

// Client is an HTTP client for interacting with the Clair API.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// ClientOption configures the Client.
type ClientOption func(*Client)

// WithHTTPClient sets the HTTP client to use for requests.
// If not provided, http.DefaultClient is used.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// NewClient creates a new Clair API client with the given base URL.
// Options can be provided to customize the client behavior.
func NewClient(baseURL string, opts ...ClientOption) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	c := &Client{
		baseURL:    u,
		httpClient: http.DefaultClient,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// CreateIndexReport submits a manifest for indexing and returns the index report.
// This corresponds to POST /indexer/api/v1/index_report.
// Expects HTTP 201 Created on success.
func (c *Client) CreateIndexReport(ctx context.Context, manifest Manifest) (*IndexReport, error) {
	reqURL := c.buildURL("/indexer/api/v1/index_report")

	body, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, c.handleError(resp)
	}

	var report IndexReport
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		return nil, fmt.Errorf("failed to decode index report: %w", err)
	}

	return &report, nil
}

// GetIndexReport retrieves an index report by manifest digest.
// This corresponds to GET /indexer/api/v1/index_report/{digest}.
// Returns ErrNotFound if the report does not exist (HTTP 404).
func (c *Client) GetIndexReport(ctx context.Context, digest string) (*IndexReport, error) {
	reqURL := c.buildURL("/indexer/api/v1/index_report/" + digest)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var report IndexReport
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		return nil, fmt.Errorf("failed to decode index report: %w", err)
	}

	return &report, nil
}

// GetVulnerabilityReport retrieves a vulnerability report by manifest digest.
// This corresponds to GET /matcher/api/v1/vulnerability_report/{digest}.
// Accepts both HTTP 200 OK and HTTP 201 Created as success.
// Returns ErrNotFound if the report does not exist (HTTP 404).
func (c *Client) GetVulnerabilityReport(ctx context.Context, digest string) (*VulnerabilityReport, error) {
	reqURL := c.buildURL("/matcher/api/v1/vulnerability_report/" + digest)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.handleError(resp)
	}

	var report VulnerabilityReport
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		return nil, fmt.Errorf("failed to decode vulnerability report: %w", err)
	}

	return &report, nil
}

// DeleteIndexReport deletes an index report by manifest digest.
// This corresponds to DELETE /indexer/api/v1/index_report/{digest}.
// Expects HTTP 204 No Content on success.
func (c *Client) DeleteIndexReport(ctx context.Context, digest string) error {
	reqURL := c.buildURL("/indexer/api/v1/index_report/" + digest)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return c.handleError(resp)
	}

	return nil
}

// GetIndexState retrieves the current state of the indexer.
// This corresponds to GET /indexer/api/v1/index_state.
func (c *Client) GetIndexState(ctx context.Context) (*IndexState, error) {
	reqURL := c.buildURL("/indexer/api/v1/index_state")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var state IndexState
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		return nil, fmt.Errorf("failed to decode index state: %w", err)
	}

	return &state, nil
}

// GetUpdateOperations retrieves the latest update operations from the matcher.
// This corresponds to GET /matcher/api/v1/internal/update_operation?latest=true.
func (c *Client) GetUpdateOperations(ctx context.Context) (map[string][]UpdateOperation, error) {
	reqURL := c.buildURL("/matcher/api/v1/internal/update_operation")

	// Add query parameter for latest operations
	u, err := url.Parse(reqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}
	q := u.Query()
	q.Set("latest", "true")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var operations map[string][]UpdateOperation
	if err := json.NewDecoder(resp.Body).Decode(&operations); err != nil {
		return nil, fmt.Errorf("failed to decode update operations: %w", err)
	}

	return operations, nil
}

// buildURL constructs a full URL by joining the base URL with the given path.
func (c *Client) buildURL(urlPath string) string {
	u := *c.baseURL
	u.Path = path.Join(u.Path, urlPath)
	return u.String()
}

// handleError reads the response body (up to 4096 bytes) and returns a formatted error.
func (c *Client) handleError(resp *http.Response) error {
	const maxErrorBytes = 4096
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBytes))
	return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
}

// DigestFromHashID extracts a Clair-compatible digest from a StackRox hash ID.
// StackRox uses "/v4/containerimage/sha256:abc..." but Clair expects "sha256:abc...".
func DigestFromHashID(hashID string) string {
	if i := strings.LastIndex(hashID, "sha256:"); i > 0 {
		return hashID[i:]
	}
	return hashID
}
