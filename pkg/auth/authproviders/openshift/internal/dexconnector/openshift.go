package dexconnector

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/storage/kubernetes/k8sapi"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/utils"
	"golang.org/x/oauth2"
)

////////////////////////////////////////////////////////////////////////////////
// The code here is based on the Dex IdP library, specifically this file:     //
//   https://github.com/dexidp/dex/blob/ff6e7c7688f363841f5cb8ffe12b41b990042f58/connector/openshift/openshift.go
//                                                                            //
// Changes include:                                                           //
//   * dynamically update oauth config with redirect_uri                      //
//   * expose access token to support refreshing of our tokens                //
//   * remove allowed groups and the corresponding validation                 //
//   * use errors.* instead of fmt.*                                          //
////////////////////////////////////////////////////////////////////////////////

const (
	openshiftWellKnownURL = "/.well-known/oauth-authorization-server"
	openshiftUsersURL     = "/apis/user.openshift.io/v1/users/~"
)

// Config holds configuration options for OpenShift OAuth login.
type Config struct {
	Issuer       string `json:"issuer"`
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
	InsecureCA   bool   `json:"insecureCA"`
	RootCA       string `json:"rootCA"`
}

type openshiftConnector struct {
	apiURL       string
	clientID     string
	clientSecret string
	cancel       context.CancelFunc
	httpClient   *http.Client
	oauth2Config *oauth2.Config
	insecureCA   bool
	rootCA       string
}

var _ connector.CallbackConnector = (*openshiftConnector)(nil)

type connectorData struct {
	// OpenShift's OAuth2 tokens expire after 24 hours while ACS tokens usually
	// after 5 minutes. We can use this token to check user attributes on ACS
	// token refresh without initiating an entire oauth flow.
	OpenShiftAccessToken string `json:"openshiftAccessToken"`
}

type user struct {
	k8sapi.TypeMeta   `json:",inline"`
	k8sapi.ObjectMeta `json:"metadata,omitempty"`
	Identities        []string `json:"identities" protobuf:"bytes,3,rep,name=identities"`
	FullName          string   `json:"fullName,omitempty" protobuf:"bytes,2,opt,name=fullName"`
	Groups            []string `json:"groups" protobuf:"bytes,4,rep,name=groups"`
}

type oauth2Error struct {
	error            string
	errorDescription string
}

func (e *oauth2Error) Error() string {
	if e.errorDescription == "" {
		return e.error
	}
	return e.error + ": " + e.errorDescription
}

// Open returns a openshiftConnector which can be used to login users through an
// upstream OpenShift OAuth2 server.
func (c *Config) Open() (*openshiftConnector, error) {
	ctx, cancel := context.WithCancel(context.Background())

	httpClient, err := newHTTPClient(c.InsecureCA, c.RootCA)
	if err != nil {
		cancel()
		return nil, errors.Wrap(err, "failed to create HTTP client")
	}

	openshiftConnector := openshiftConnector{
		apiURL:       c.Issuer,
		clientID:     c.ClientID,
		clientSecret: c.ClientSecret,
		cancel:       cancel,
		httpClient:   httpClient,
		insecureCA:   c.InsecureCA,
		rootCA:       c.RootCA,
	}

	// Discover information about the OAuth server.
	wellKnownURL := strings.TrimSuffix(c.Issuer, "/") + openshiftWellKnownURL
	req, err := http.NewRequest(http.MethodGet, wellKnownURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "creating a well-known request")
	}

	var metadata struct {
		Auth  string `json:"authorization_endpoint"`
		Token string `json:"token_endpoint"`
	}

	resp, err := openshiftConnector.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		cancel()
		return nil, errors.Wrap(err, "failed to query OpenShift endpoint")
	}

	defer utils.IgnoreError(resp.Body.Close)

	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		cancel()
		return nil, errors.Wrapf(err, "discovery through endpoint %q failed to decode body", wellKnownURL)
	}

	openshiftConnector.oauth2Config = &oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		Scopes:       []string{"user:info"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  metadata.Auth,
			TokenURL: metadata.Token,
		},
	}

	return &openshiftConnector, nil
}

func (c *openshiftConnector) Close() error {
	c.cancel()
	return nil
}

// LoginURL returns the URL to redirect the user to login with.
func (c *openshiftConnector) LoginURL(_ connector.Scopes, callbackURL string, state string) (string, error) {
	clonedConfig := *c.oauth2Config
	clonedConfig.RedirectURL = callbackURL
	return clonedConfig.AuthCodeURL(state, oauth2.SetAuthURLParam("redirect_uri", callbackURL)), nil
}

// HandleCallback parses the request and returns the user's identity.
func (c *openshiftConnector) HandleCallback(s connector.Scopes, r *http.Request) (identity connector.Identity, err error) {
	q := r.URL.Query()
	if errType := q.Get("error"); errType != "" {
		return identity, &oauth2Error{errType, q.Get("error_description")}
	}

	ctx := r.Context()
	if c.httpClient != nil {
		ctx = context.WithValue(r.Context(), oauth2.HTTPClient, c.httpClient)
	}

	token, err := c.oauth2Config.Exchange(ctx, q.Get("code"))
	if err != nil {
		return identity, errors.Wrap(err, "failed to get token")
	}

	client := c.oauth2Config.Client(ctx, token)
	user, err := c.user(ctx, client)
	if err != nil {
		return identity, errors.Wrap(err, "openshift: get user")
	}

	identity = connector.Identity{
		UserID:            user.UID,
		Username:          user.Name,
		PreferredUsername: user.Name,
		Email:             user.Name,
		Groups:            user.Groups,
	}

	if s.OfflineAccess {
		data := connectorData{OpenShiftAccessToken: token.AccessToken}
		connData, err := json.Marshal(data)
		if err != nil {
			return identity, errors.Wrap(err, "failed to marshal openshift's access token")
		}
		identity.ConnectorData = connData
	}

	return identity, nil
}

// user returns the OpenShift user associated with the token passed in client.
func (c *openshiftConnector) user(ctx context.Context, client *http.Client) (u user, err error) {
	url := strings.TrimSuffix(c.apiURL, "/") + openshiftUsersURL

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return u, errors.Wrap(err, "creating a users request")
	}

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return u, errors.Wrapf(err, "querying %q", openshiftUsersURL)
	}
	defer utils.IgnoreError(resp.Body.Close)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return u, errors.Wrap(err, "reading response body")
	}
	if resp.StatusCode != http.StatusOK {
		return u, errors.Errorf("%s: %s", resp.Status, body)
	}

	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&u); err != nil {
		return u, errors.Wrap(err, "decode JSON body")
	}

	return u, err
}

// newHTTPClient returns a new HTTP client.
func newHTTPClient(insecureCA bool, rootCA string) (*http.Client, error) {
	tlsConfig := tls.Config{}

	if insecureCA {
		tlsConfig = tls.Config{InsecureSkipVerify: true}
	} else if rootCA != "" {
		tlsConfig = tls.Config{RootCAs: x509.NewCertPool()}
		rootCABytes, err := os.ReadFile(rootCA)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read root-ca")
		}
		if !tlsConfig.RootCAs.AppendCertsFromPEM(rootCABytes) {
			return nil, errors.Errorf("no certs found in root CA file %q", rootCA)
		}
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tlsConfig,
			Proxy:           http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}, nil
}
