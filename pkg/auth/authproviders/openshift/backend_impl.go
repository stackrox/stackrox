package openshift

import (
	"context"
	"crypto/x509"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/authproviders/idputil"
	"github.com/stackrox/rox/pkg/auth/authproviders/openshift/internal/dexconnector"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/satoken"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	openshiftAPIUrl    = "https://openshift.default.svc"
	roxTokenExpiration = 5 * time.Minute
)

// This is the location for CA files which shall be used for certificate validation within
// openshift auth. In addition to the CA files here, the system's trusted root CAs will be used as well.
// The path may or may not exist depending on cluster state & configuration.
const (
	// serviceOperatorCAPath points to the secret of the service account, which within an OpenShift environment
	// also has the service-ca.crt, which includes the CA to verify certificates issued by the service-ca operator.
	// This could be i.e. the default ingress controller certificate.
	serviceOperatorCAPath = "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"
	// internalServicesCAPath points to the secret of the service account, which includes the internal CAs to
	// verify internal cluster services.
	// This could be i.e. the openshiftAPIUrl or other internal services.
	internalServicesCAPath = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	// injectedCAPath points to the bundle of user-provided and system CA certificates
	// merged by the Cluster Network Operator.
	injectedCAPath = "/etc/pki/injected-ca-trust/tls-ca-bundle.pem"
)

var (
	defaultScopes = dexconnector.Scopes{
		OfflineAccess: true,
		Groups:        true,
	}
)

type callbackAndRefreshConnector interface {
	dexconnector.CallbackConnector
	dexconnector.RefreshConnector
}

type backend struct {
	id                      string
	baseRedirectURLPath     string
	openshiftConnector      callbackAndRefreshConnector
	openshiftConnectorMutex sync.Mutex
}

type openShiftSettings struct {
	clientID        string
	clientSecret    string
	trustedCertPool *x509.CertPool
}

var _ authproviders.RefreshTokenEnabledBackend = (*backend)(nil)

func newBackend(id string, callbackURLPath string, _ map[string]string) (authproviders.Backend, error) {
	openshiftConnector, err := createOpenshiftConnector()
	if err != nil {
		return nil, err
	}

	b := &backend{
		id:                  id,
		baseRedirectURLPath: callbackURLPath,
		openshiftConnector:  openshiftConnector,
	}

	// Start watching the underlying cert pool injected into the openshift connector.
	// In case the cert pool changes, we re-create the openshift connector so that newly added trusted CAs
	// are being added included.
	watchCertPool(b.recreateOpenshiftConnector)

	return b, nil
}

func createOpenshiftConnector() (callbackAndRefreshConnector, error) {
	settings, err := getOpenShiftSettings()
	if err != nil {
		return nil, err
	}

	dexCfg := dexconnector.Config{
		Issuer:          openshiftAPIUrl,
		ClientID:        settings.clientID,
		ClientSecret:    settings.clientSecret,
		TrustedCertPool: settings.trustedCertPool,
	}

	openshiftConnector, err := dexCfg.Open()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create dex openshiftConnector for OpenShift's OAuth Server")
	}

	return openshiftConnector, nil
}

// There is no config but static settings instead.
func (b *backend) Config() map[string]string {
	return nil
}

func (b *backend) LoginURL(clientState string, ri *requestinfo.RequestInfo) (string, error) {
	state := idputil.MakeState(b.id, clientState)

	// Augment baseRedirectURLPath to a redirect URL with hostname, etc set.
	redirectURI := dexconnector.MakeRedirectURI(ri, b.baseRedirectURLPath)

	return b.openshiftConnector.LoginURL(defaultScopes, redirectURI.String(), state)
}

func (b *backend) RefreshURL() string {
	return ""
}

func (b *backend) OnEnable(_ authproviders.Provider) {}

func (b *backend) OnDisable(_ authproviders.Provider) {}

func (b *backend) ProcessHTTPRequest(_ http.ResponseWriter, r *http.Request) (*authproviders.AuthResponse, error) {
	if r.URL.Path != b.baseRedirectURLPath {
		return nil, httputil.Errorf(http.StatusNotFound, "path %q not found", r.URL.Path)
	}
	if r.Method != http.MethodGet {
		return nil, httputil.Errorf(http.StatusMethodNotAllowed, "unsupported method %q, only GET requests are allowed to this URL", r.Method)
	}

	id, err := b.openshiftConnector.HandleCallback(defaultScopes, r)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving user identity")
	}
	return b.idToAuthResponse(&id), nil
}

func (b *backend) idToAuthResponse(id *dexconnector.Identity) *authproviders.AuthResponse {
	// OpenShift doesn't provide emails in their users API response, see
	// https://docs.openshift.com/container-platform/4.9/rest_api/user_and_group_apis/user-user-openshift-io-v1.html
	attributes := map[string][]string{
		authproviders.UseridAttribute: {string(id.UserID)},
		authproviders.NameAttribute:   {id.Username},
		authproviders.GroupsAttribute: id.Groups,
	}

	return &authproviders.AuthResponse{
		Claims: &tokens.ExternalUserClaim{
			UserID:     id.Username,
			FullName:   id.Username,
			Attributes: attributes,
		},
		Expiration: time.Now().Add(roxTokenExpiration),
		RefreshTokenData: authproviders.RefreshTokenData{
			RefreshToken: string(id.ConnectorData),
		},
	}
}

// RefreshAccessToken attempts to fetch user info and issue an updated auth
// status. If the refresh token has expired, error is returned.
func (b *backend) RefreshAccessToken(ctx context.Context, refreshTokenData authproviders.RefreshTokenData) (*authproviders.AuthResponse, error) {
	id, err := b.openshiftConnector.Refresh(ctx, defaultScopes, dexconnector.Identity{
		ConnectorData: []byte(refreshTokenData.RefreshToken),
	})
	if err != nil {
		return nil, errors.Wrap(err, "retrieving user identity")
	}
	return b.idToAuthResponse(&id), nil
}

func (b *backend) RevokeRefreshToken(_ context.Context, _ authproviders.RefreshTokenData) error {
	return nil
}

func (b *backend) ExchangeToken(_ context.Context, _ string, _ string) (*authproviders.AuthResponse, string, error) {
	return nil, "", errors.New("not implemented")
}

func (b *backend) Validate(_ context.Context, _ *tokens.Claims) error {
	return nil
}

func (b *backend) recreateOpenshiftConnector() {
	openshiftConnector, err := createOpenshiftConnector()
	if err != nil {
		log.Errorw("failed to create updated dex openshiftConnector for OpenShift's OAuth Server with new CAs, "+
			"new certs will not be applied. This may lead to unwanted TLS connection issues.", logging.Err(err))
		return
	}

	b.openshiftConnectorMutex.Lock()
	defer b.openshiftConnectorMutex.Unlock()
	b.openshiftConnector = openshiftConnector
}

func getOpenShiftSettings() (openShiftSettings, error) {
	clientID := "system:serviceaccount:" + env.Namespace.Setting() + ":central"

	clientSecret, err := satoken.LoadTokenFromFile()
	if err != nil {
		return openShiftSettings{}, errors.Wrap(err, "reading service account token")
	}

	certPool, err := getSystemCertPoolWithAdditionalCA(serviceOperatorCAPath, internalServicesCAPath, injectedCAPath)
	if err != nil {
		return openShiftSettings{}, err
	}

	return openShiftSettings{
		clientID:        clientID,
		clientSecret:    clientSecret,
		trustedCertPool: certPool,
	}, nil
}

func getSystemCertPoolWithAdditionalCA(additionalCAPaths ...string) (*x509.CertPool, error) {
	// Use the x509.SystemCertPool to include system's trusted CAs.
	sysCertPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, errors.Wrap(err, "creating system cert pool")
	}

	sysAndAdditionalCertsPool, err := addAdditionalCAsToCertPool(additionalCAPaths, sysCertPool)
	if err != nil {
		return nil, err
	}

	return sysAndAdditionalCertsPool, nil
}

func addAdditionalCAsToCertPool(additionalCAPaths []string, certPool *x509.CertPool) (*x509.CertPool, error) {
	for _, caPath := range additionalCAPaths {
		rootCABytes, exists, err := readCA(caPath)
		if !exists {
			continue
		}
		if err != nil {
			return nil, errors.Wrapf(err, "reading CA at path %s", caPath)
		}
		if !certPool.AppendCertsFromPEM(rootCABytes) {
			return nil, errors.Errorf("parsing root CA file from %s", caPath)
		}
	}
	return certPool, nil
}
