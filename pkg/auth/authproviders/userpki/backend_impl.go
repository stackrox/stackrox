package userpki

import (
	"context"
	"crypto/x509"
	"net/http"
	"strings"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/cryptoutils"
	"github.com/stackrox/rox/pkg/grpc/authn/userpki"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// ConfigKeys is the map key in the provider configuration
	ConfigKeys = "keys"

	authenticateHandlerPath = "authenticate"
)

var (
	log                   = logging.LoggerForModule()
	errNoCertificate      = errors.New("user certificate not present")
	errInvalidCertificate = errors.New("user certificate doesn't match any configured provider")
)

func newBackend(_ context.Context, pathPrefix string, callbacks ProviderCallbacks, config map[string]string) (authproviders.Backend, error) {
	pem := config[ConfigKeys]
	if pem == "" {
		return nil, errors.Errorf("parameter %q is required", ConfigKeys)
	}
	certs, err := helpers.ParseCertificatesPEM([]byte(pem))
	if err != nil {
		return nil, err
	}
	fingerprints := set.NewStringSet()
	for _, cert := range certs {
		fingerprints.Add(cryptoutils.CertFingerprint(cert))
	}
	return &backendImpl{
		pathPrefix:   pathPrefix,
		callbacks:    callbacks,
		certs:        certs,
		fingerprints: fingerprints,
		config: map[string]string{
			ConfigKeys: pem,
		},
	}, nil
}

type backendImpl struct {
	pathPrefix   string
	callbacks    ProviderCallbacks
	certs        []*x509.Certificate
	fingerprints set.StringSet
	config       map[string]string
}

func (p *backendImpl) Config() map[string]string {
	return p.config
}

func (p *backendImpl) OnEnable(provider authproviders.Provider) {
	log.Debugf("Provider %q enabled", provider.ID())
	p.callbacks.RegisterAuthProvider(provider, p.certs)
}

func (p *backendImpl) OnDisable(provider authproviders.Provider) {
	log.Debugf("Provider %q disabled", provider.ID())
	p.callbacks.UnregisterAuthProvider(provider)
}

func (p *backendImpl) LoginURL(_ string, _ *requestinfo.RequestInfo) (string, error) {
	return p.pathPrefix + authenticateHandlerPath, nil
}

func (p *backendImpl) RefreshURL() string {
	return ""
}

func (p *backendImpl) ProcessHTTPRequest(_ http.ResponseWriter, r *http.Request) (*authproviders.AuthResponse, error) {
	restPath := strings.TrimPrefix(r.URL.Path, p.pathPrefix)
	if len(restPath) == len(r.URL.Path) {
		return nil, utils.ShouldErr(httputil.Errorf(http.StatusNotFound,
			"invalid URL %q, expected sub-path of %q", r.URL.Path, p.pathPrefix))
	}

	if restPath != authenticateHandlerPath {
		log.Debugf("Invalid REST path %q", restPath)
		return nil, httputil.NewError(http.StatusNotFound, "Not Found")
	}

	if r.Method != http.MethodGet {
		return nil, httputil.Errorf(http.StatusMethodNotAllowed,
			"invalid method %q, only GET requests are allowed", r.Method)
	}

	ri := requestinfo.FromContext(r.Context())
	if len(ri.VerifiedChains) == 0 {
		return nil, errNoCertificate
	}

	for _, chain := range ri.VerifiedChains {
		valid := false
		for i := len(chain) - 1; i > 0; i-- {
			if p.fingerprints.Contains(chain[i].CertFingerprint) {
				valid = true
				break
			}
		}
		if !valid {
			continue
		}

		userCert := chain[0]
		authResp := &authproviders.AuthResponse{
			Claims:     externalUser(userCert),
			Expiration: userCert.NotAfter,
		}
		return authResp, nil
	}

	return nil, errInvalidCertificate
}

func (p *backendImpl) ExchangeToken(_ context.Context, _, _ string) (*authproviders.AuthResponse, string, error) {
	return nil, "", status.Errorf(codes.Unimplemented, "ExchangeToken not implemented for provider type %q", TypeName)
}

func (p *backendImpl) Validate(ctx context.Context, claims *tokens.Claims) error {
	ri := requestinfo.FromContext(ctx)
	if len(ri.VerifiedChains) == 0 || len(ri.VerifiedChains[0]) == 0 {
		return errors.New("No client chains present")
	}
	if userID(ri.VerifiedChains[0][0]) != claims.ExternalUser.UserID {
		return errors.New("Certificate fingerprint changed, please log in again")
	}
	return nil
}

func userID(info mtls.CertInfo) string {
	return "userpki:" + info.CertFingerprint
}

func externalUser(info mtls.CertInfo) *tokens.ExternalUserClaim {
	attrs := userpki.ExtractAttributes(info)
	return &tokens.ExternalUserClaim{
		UserID:     userID(info),
		FullName:   info.Subject.CommonName,
		Attributes: attrs,
	}
}
