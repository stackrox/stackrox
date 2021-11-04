package openshift

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/dexidp/dex/connector"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/authproviders/idputil"
	"github.com/stackrox/rox/pkg/auth/authproviders/openshift/internal/dexconnector"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/satoken"
)

const (
	openshiftAPIUrl = "https://openshift.default.svc"

	roxTokenExpiration = 5 * time.Minute
)

var (
	defaultScopes = connector.Scopes{
		OfflineAccess: true,
		Groups:        true,
	}
)

type backend struct {
	id                 string
	baseRedirectURL    url.URL
	openshiftConnector connector.CallbackConnector
}

var _ authproviders.Backend = (*backend)(nil)

func newBackend(id string, callbackURLPath string, _ map[string]string) (authproviders.Backend, error) {
	clientID, clientSecret, err := openshiftSettings()
	if err != nil {
		return nil, err
	}

	baseRedirectURL := url.URL{
		Scheme: "https",
		Path:   callbackURLPath,
	}

	dexCfg := dexconnector.Config{
		Issuer:       openshiftAPIUrl,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		// TODO(ROX-8099): Do not skip server cert verification.
		InsecureCA: true,
	}

	openshiftConnector, err := dexCfg.Open()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create dex openshiftConnector for OpenShift's OAuth Server")
	}

	b := &backend{
		id:                 id,
		baseRedirectURL:    baseRedirectURL,
		openshiftConnector: openshiftConnector,
	}

	return b, nil
}

// There is no config but static settings instead.
func (b *backend) Config() map[string]string {
	return nil
}

func (b *backend) LoginURL(clientState string, ri *requestinfo.RequestInfo) string {
	state := idputil.MakeState(b.id, clientState)

	// baseRedirectURL does not include the hostname, take it from the request.
	// Allow HTTP only if the client did not use TLS and the host is localhost.
	redirectURL := b.baseRedirectURL
	redirectURL.Host = ri.Hostname
	if !ri.ClientUsedTLS && netutil.IsLocalEndpoint(redirectURL.Host) {
		redirectURL.Scheme = "http"
	}

	loginURL, _ := b.openshiftConnector.LoginURL(defaultScopes, redirectURL.String(), state)
	return loginURL
}

func (b *backend) RefreshURL() string {
	return ""
}

func (b *backend) OnEnable(provider authproviders.Provider) {}

func (b *backend) OnDisable(provider authproviders.Provider) {}

func (b *backend) ProcessHTTPRequest(_ http.ResponseWriter, r *http.Request) (*authproviders.AuthResponse, error) {
	if r.URL.Path != b.baseRedirectURL.Path {
		return nil, httputil.Errorf(http.StatusNotFound, "path %q not found", r.URL.Path)
	}
	if r.Method != http.MethodGet {
		return nil, httputil.Errorf(http.StatusMethodNotAllowed, "unsupported method %q, only GET requests are allowed to this URL", r.Method)
	}
	id, err := b.openshiftConnector.HandleCallback(defaultScopes, r)
	if err != nil {
		return nil, err
	}
	return &authproviders.AuthResponse{
		Claims: &tokens.ExternalUserClaim{
			UserID: id.Username,
			Email:  id.Email,
			Attributes: map[string][]string{
				"groups": id.Groups,
			},
		},
		Expiration: time.Now().Add(roxTokenExpiration),
	}, nil
}

func (b *backend) ExchangeToken(ctx context.Context, externalToken string, state string) (*authproviders.AuthResponse, string, error) {
	return nil, "", errors.New("not implemented")
}

func (b *backend) Validate(_ context.Context, _ *tokens.Claims) error {
	return nil
}

func openshiftSettings() (string, string, error) {
	clientID := "system:serviceaccount:" + env.Namespace.Setting() + ":central"

	clientSecret, err := satoken.LoadTokenFromFile()
	if err != nil {
		return "", "", errors.Wrap(err, "reading service account token")
	}

	return clientID, clientSecret, nil
}
