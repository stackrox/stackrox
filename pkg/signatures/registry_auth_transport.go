package signatures

// Inline registry auth transport replacing heroku/docker-registry-client.
// Provides an http.RoundTripper that handles Docker Registry V2 bearer token
// and basic auth, wrapping HTTP errors as httpStatusError for error matching.

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// httpStatusError wraps HTTP error responses with the response object.
// This replaces registry.HttpStatusError from heroku/docker-registry-client.
type httpStatusError struct {
	Response *http.Response
	Body     []byte
}

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("http: non-successful response (status=%v body=%q)", e.Response.StatusCode, e.Body)
}

// wrapTransport creates an http.RoundTripper that authenticates with Docker
// registries using Bearer token and Basic auth. This replaces
// registry.WrapTransport from heroku/docker-registry-client.
func wrapTransport(base http.RoundTripper, registryURL, username, password string) http.RoundTripper {
	return &errorTransport{
		base: &basicAuthTransport{
			base: &tokenTransport{
				base:     base,
				username: username,
				password: password,
			},
			url:      registryURL,
			username: username,
			password: password,
		},
	}
}

// tokenTransport handles Bearer token authentication.
// On 401, it parses the Www-Authenticate challenge and fetches a token.
type tokenTransport struct {
	base     http.RoundTripper
	username string
	password string
	token    string
}

func (t *tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}

	// Parse Bearer challenge and fetch token
	authService := parseBearerChallenge(resp)
	if authService == nil {
		return resp, nil
	}
	if resp.Body != nil {
		resp.Body.Close()
	}

	token, authResp, err := t.fetchToken(authService)
	if err != nil {
		return authResp, err
	}
	if authResp != nil && authResp.Body != nil {
		authResp.Body.Close()
	}

	t.token = token
	req.Header.Set("Authorization", "Bearer "+token)
	return t.base.RoundTrip(req)
}

func (t *tokenTransport) fetchToken(auth *bearerChallenge) (string, *http.Response, error) {
	tokenURL := fmt.Sprintf("%s?service=%s", auth.realm, auth.service)
	if auth.scope != "" {
		tokenURL += "&scope=" + auth.scope
	}
	tokenReq, err := http.NewRequest("GET", tokenURL, nil)
	if err != nil {
		return "", nil, err
	}
	if t.username != "" || t.password != "" {
		tokenReq.SetBasicAuth(t.username, t.password)
	}

	resp, err := t.base.RoundTrip(tokenReq)
	if err != nil {
		return "", nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return "", resp, nil
	}
	defer resp.Body.Close()

	var tokenResp struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", nil, err
	}

	token := tokenResp.Token
	if token == "" {
		token = tokenResp.AccessToken
	}
	return token, nil, nil
}

type bearerChallenge struct {
	realm   string
	service string
	scope   string
}

// parseBearerChallenge extracts realm/service/scope from a 401 response.
func parseBearerChallenge(resp *http.Response) *bearerChallenge {
	wwwAuth := resp.Header.Get("Www-Authenticate")
	if !strings.HasPrefix(strings.ToLower(wwwAuth), "bearer ") {
		return nil
	}
	params := parseAuthParams(wwwAuth[7:])
	if params["realm"] == "" {
		return nil
	}
	return &bearerChallenge{
		realm:   params["realm"],
		service: params["service"],
		scope:   params["scope"],
	}
}

// parseAuthParams parses key="value" pairs from a Www-Authenticate header.
func parseAuthParams(s string) map[string]string {
	result := make(map[string]string)
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		eq := strings.IndexByte(part, '=')
		if eq < 0 {
			continue
		}
		key := strings.ToLower(part[:eq])
		value := strings.Trim(part[eq+1:], `"`)
		result[key] = value
	}
	return result
}

// basicAuthTransport adds Basic auth headers for requests matching the registry URL.
type basicAuthTransport struct {
	base     http.RoundTripper
	url      string
	username string
	password string
}

func (t *basicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.HasPrefix(req.URL.String(), t.url) && (t.username != "" || t.password != "") {
		req.SetBasicAuth(t.username, t.password)
	}
	return t.base.RoundTrip(req)
}

// errorTransport wraps HTTP 4xx+ responses as httpStatusError.
type errorTransport struct {
	base http.RoundTripper
}

func (t *errorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("http: failed to read response body (status=%v, err=%q)", resp.StatusCode, err)
		}
		return nil, &httpStatusError{Response: resp, Body: body}
	}
	return resp, nil
}
