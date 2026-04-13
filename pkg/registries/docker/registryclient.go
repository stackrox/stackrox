package docker

// Lightweight Docker Registry V2 HTTP client.
// Replaces heroku/docker-registry-client (198 deps) with ~80 lines.
//
// Handles:
// - Basic auth and Bearer token auth (challenge-response)
// - Manifest fetch (GET /v2/<repo>/manifests/<ref>)
// - Repository listing (GET /v2/_catalog)
// - Error wrapping with HTTP status codes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

// registryClient is a minimal Docker Registry V2 HTTP client.
type registryClient struct {
	url       string
	username  string
	password  string
	transport http.RoundTripper
	token     string // cached bearer token
}

func newRegistryClient(url string, username, password string, transport http.RoundTripper) *registryClient {
	return &registryClient{
		url:       strings.TrimRight(url, "/"),
		username:  username,
		password:  password,
		transport: transport,
	}
}

// registryClientError wraps HTTP errors with the status code.
type registryClientError struct {
	StatusCode int
	Message    string
}

func (e *registryClientError) Error() string {
	return fmt.Sprintf("registry error %d: %s", e.StatusCode, e.Message)
}

// do performs an authenticated request, handling Bearer token auth if needed.
func (c *registryClient) do(ctx context.Context, method, path string, accept string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.url+path, nil)
	if err != nil {
		return nil, err
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}

	// Try with existing auth
	c.addAuth(req)
	resp, err := c.transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// If 401, try Bearer token auth
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		if err := c.fetchBearerToken(ctx, resp.Header.Get("Www-Authenticate")); err != nil {
			return nil, err
		}
		// Retry with token
		req, _ = http.NewRequestWithContext(ctx, method, c.url+path, nil)
		if accept != "" {
			req.Header.Set("Accept", accept)
		}
		c.addAuth(req)
		resp, err = c.transport.RoundTrip(req)
		if err != nil {
			return nil, err
		}
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		resp.Body.Close()
		return nil, &registryClientError{StatusCode: resp.StatusCode, Message: string(body)}
	}

	return resp, nil
}

func (c *registryClient) addAuth(req *http.Request) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	} else if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
}

// fetchBearerToken parses the Www-Authenticate header and fetches a Bearer token.
func (c *registryClient) fetchBearerToken(ctx context.Context, wwwAuth string) error {
	// Parse: Bearer realm="https://auth.docker.io/token",service="registry.docker.io",scope="repository:library/nginx:pull"
	if !strings.HasPrefix(wwwAuth, "Bearer ") {
		return nil // Not Bearer auth
	}

	params := parseWWWAuth(wwwAuth[7:])
	realm := params["realm"]
	if realm == "" {
		return errors.New("no realm in Bearer challenge")
	}

	tokenURL := fmt.Sprintf("%s?service=%s&scope=%s", realm, params["service"], params["scope"])
	req, _ := http.NewRequestWithContext(ctx, "GET", tokenURL, nil)
	if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.transport.RoundTrip(req)
	if err != nil {
		return errors.Wrap(err, "fetching bearer token")
	}
	defer resp.Body.Close()

	var tokenResp struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return errors.Wrap(err, "decoding token response")
	}

	c.token = tokenResp.Token
	if c.token == "" {
		c.token = tokenResp.AccessToken
	}
	return nil
}

// parseWWWAuth parses key="value" pairs from a Www-Authenticate header value.
func parseWWWAuth(s string) map[string]string {
	result := make(map[string]string)
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		eq := strings.IndexByte(part, '=')
		if eq < 0 {
			continue
		}
		key := part[:eq]
		value := strings.Trim(part[eq+1:], `"`)
		result[key] = value
	}
	return result
}
