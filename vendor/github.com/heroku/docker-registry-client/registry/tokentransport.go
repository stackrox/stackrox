package registry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
)

type TokenTransport struct {
	Transport http.RoundTripper
	Username  string
	Password  string

	token      string
	tokenMutex sync.RWMutex
}

func (t *TokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.Transport.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	if authService := isTokenDemand(resp); authService != nil {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		resp, err = t.authAndRetry(authService, req)
	}
	return resp, err
}

type authToken struct {
	Token       string `json:"token"`
	AccessToken string `json:"access_token"`
}

func (a authToken) getToken() string {
	if a.Token != "" {
		return a.Token
	}
	return a.AccessToken
}

func (t *TokenTransport) authAndRetry(authService *authService, req *http.Request) (*http.Response, error) {
	token, authResp, err := t.auth(authService)
	if err != nil {
		return authResp, err
	}

	if authResp != nil && authResp.Body != nil {
		_ = authResp.Body.Close()
	}
	retryResp, err := t.retry(req, token)
	return retryResp, err
}

func (t *TokenTransport) auth(authService *authService) (string, *http.Response, error) {
	authReq, err := authService.Request(t.Username, t.Password)
	if err != nil {
		return "", nil, err
	}

	client := http.Client{
		Transport: t.Transport,
	}

	response, err := client.Do(authReq)
	if err != nil {
		return "", nil, err
	}

	if response.StatusCode != http.StatusOK {
		return "", response, err
	}
	defer response.Body.Close()

	var authToken authToken
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&authToken)
	if err != nil {
		return "", nil, err
	}

	t.tokenMutex.Lock()
	defer t.tokenMutex.Unlock()
	t.token = authToken.getToken()
	return t.token, nil, nil
}

func (t *TokenTransport) retry(req *http.Request, token string) (*http.Response, error) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	resp, err := t.Transport.RoundTrip(req)
	return resp, err
}

// GetToken returns the current token used to access the registry
func (t *TokenTransport) GetToken() string {
	t.tokenMutex.RLock()
	defer t.tokenMutex.RUnlock()

	return t.token
}

type authService struct {
	Realm   string
	Service string
	Scope   string
}

func (authService *authService) Request(username, password string) (*http.Request, error) {
	url, err := url.Parse(authService.Realm)
	if err != nil {
		return nil, err
	}

	q := url.Query()
	q.Set("service", authService.Service)
	if authService.Scope != "" {
		q.Set("scope", authService.Scope)
	}
	url.RawQuery = q.Encode()

	request, err := http.NewRequest("GET", url.String(), nil)

	if username != "" || password != "" {
		request.SetBasicAuth(username, password)
	}

	return request, err
}

func isTokenDemand(resp *http.Response) *authService {
	if resp == nil {
		return nil
	}
	if resp.StatusCode != http.StatusUnauthorized {
		return nil
	}
	return parseOauthHeader(resp)
}

func parseOauthHeader(resp *http.Response) *authService {
	challenges := parseAuthHeader(resp.Header)
	for _, challenge := range challenges {
		if challenge.Scheme == "bearer" {
			return &authService{
				Realm:   challenge.Parameters["realm"],
				Service: challenge.Parameters["service"],
				Scope:   challenge.Parameters["scope"],
			}
		}
	}
	return nil
}
