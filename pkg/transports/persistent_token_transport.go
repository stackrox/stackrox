package transports

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/httputil"
	"github.com/stackrox/stackrox/pkg/utils"
)

const (
	timeout = 10 * time.Second
)

// PersistentTokenTransport is a transport that can be used to retrieve a token once from a registry and then be reused
type PersistentTokenTransport struct {
	Transport http.RoundTripper
	Username  string
	Password  string
	Registry  string
	Token     string
}

// NewPersistentTokenTransport returns a new transport or an error if the token could not be generated
func NewPersistentTokenTransport(registry, username, password string) (*PersistentTokenTransport, error) {
	tran := &PersistentTokenTransport{
		Transport: http.DefaultTransport,
		Username:  username,
		Password:  password,
		Registry:  registry,
	}
	if err := tran.refreshToken(); err != nil {
		return nil, err
	}
	return tran, nil
}

type tokenResp struct {
	Token string `json:"token"`
}

func (t *PersistentTokenTransport) refreshToken() error {
	req, err := http.NewRequest("GET", t.Registry+"/v2/token", nil)
	if err != nil {
		return err
	}

	client := http.Client{
		Timeout: timeout,
	}
	req.SetBasicAuth(t.Username, t.Password)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if !httputil.Is2xxStatusCode(resp.StatusCode) {
		return errors.New(resp.Status)
	}

	defer utils.IgnoreError(resp.Body.Close)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var tokenResp tokenResp
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return err
	}

	t.Token = tokenResp.Token
	return nil
}

func (t *PersistentTokenTransport) setToken(req *http.Request) {
	req.Header.Add("Authorization", "Bearer "+t.Token)
}

// RoundTrip implements the roundtripper interface and will try to refresh the token if it has expired
func (t *PersistentTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.setToken(req)
	resp, err := t.Transport.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	if resp.StatusCode != http.StatusUnauthorized {
		return resp, err
	} else if err := t.refreshToken(); err != nil {
		return resp, err
	}
	t.setToken(req)
	return t.Transport.RoundTrip(req)
}

// GetToken returns the auth token
func (t *PersistentTokenTransport) GetToken() string {
	return t.Token
}
