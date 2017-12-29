package tenable

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type persistentTokenTransport struct {
	Transport http.RoundTripper
	Username  string
	Password  string
	Registry  string
	Token     string
}

func newPersistentTokenTransport(registry, username, password string) (*persistentTokenTransport, error) {
	tran := &persistentTokenTransport{
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

func (t *persistentTokenTransport) refreshToken() error {
	req, err := http.NewRequest("GET", t.Registry+"/v2/token", nil)
	if err != nil {
		return err
	}
	var client http.Client
	req.SetBasicAuth(t.Username, t.Password)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
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

func (t *persistentTokenTransport) setToken(req *http.Request) {
	req.Header.Add("Authorization", "Bearer "+t.Token)
}

func (t *persistentTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
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
