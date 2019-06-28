package userpki

import (
	"context"
	"crypto/x509"
	"fmt"
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
	"github.com/stackrox/rox/pkg/set"
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

func newBackend(ctx context.Context, pathPrefix string, callbacks ProviderCallbacks, config map[string]string) (authproviders.Backend, map[string]string, error) {
	pem := config[ConfigKeys]
	if pem == "" {
		return nil, nil, fmt.Errorf("parameter %q is required", ConfigKeys)
	}
	certs, err := helpers.ParseCertificatesPEM([]byte(pem))
	if err != nil {
		return nil, nil, err
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
	}, config, nil
}

type backendImpl struct {
	pathPrefix   string
	callbacks    ProviderCallbacks
	certs        []*x509.Certificate
	fingerprints set.StringSet
}

func (p *backendImpl) OnEnable(provider authproviders.Provider) {
	log.Debugf("Provider %q enabled", provider.ID())
	p.callbacks.RegisterAuthProvider(provider, p.certs)
}

func (p *backendImpl) OnDisable(provider authproviders.Provider) {
	log.Debugf("Provider %q disabled", provider.ID())
	p.callbacks.UnregisterAuthProvider(provider)
}

func (p *backendImpl) LoginURL(clientState string, ri *requestinfo.RequestInfo) string {
	return p.pathPrefix + authenticateHandlerPath
}

func (p *backendImpl) RefreshURL() string {
	return ""
}

func (p *backendImpl) ProcessHTTPRequest(w http.ResponseWriter, r *http.Request) (*tokens.ExternalUserClaim, []tokens.Option, string, error) {
	restPath := strings.TrimPrefix(r.URL.Path, p.pathPrefix)
	if len(restPath) == len(r.URL.Path) {
		log.Debugf("Invalid URL %q wrt %q", r.URL.Path, p.pathPrefix)
		return nil, nil, "", httputil.NewError(http.StatusNotFound, "Not Found")
	}

	if restPath != authenticateHandlerPath {
		log.Debugf("Invalid REST path %q", restPath)
		return nil, nil, "", httputil.NewError(http.StatusNotFound, "Not Found")
	}
	if r.Method != http.MethodGet {
		return nil, nil, "", httputil.NewError(http.StatusMethodNotAllowed, "Method Not Allowed")
	}
	ri := requestinfo.FromContext(r.Context())
	if len(ri.VerifiedChains) != 1 {
		return nil, nil, "", errNoCertificate
	}
	for _, ca := range ri.VerifiedChains[0] {
		if p.fingerprints.Contains(ca.CertFingerprint) {
			continue
		}
		userCert := ri.VerifiedChains[0][0]
		return externalUser(userCert), options(userCert), "", nil
	}
	return nil, nil, "", errInvalidCertificate
}

func (p *backendImpl) ExchangeToken(ctx context.Context, externalToken, state string) (*tokens.ExternalUserClaim, []tokens.Option, string, error) {
	return nil, nil, "", status.Errorf(codes.Unimplemented, "ExchangeToken not implemented for provider type %q", TypeName)
}

func (p *backendImpl) Validate(ctx context.Context, claims *tokens.Claims) error {
	ri := requestinfo.FromContext(ctx)
	if len(ri.VerifiedChains) != 1 || len(ri.VerifiedChains[0]) == 0 {
		return errors.New("No client chains present")
	}
	if userID(ri.VerifiedChains[0][0]) != claims.ExternalUser.UserID {
		return errors.New("Certificate fingerprint changed, please log in again")
	}
	return nil
}

func userID(info requestinfo.CertInfo) string {
	return "userpki:" + info.CertFingerprint
}

func externalUser(info requestinfo.CertInfo) *tokens.ExternalUserClaim {
	attrs := userpki.ExtractAttributes(info)
	return &tokens.ExternalUserClaim{
		UserID:     userID(info),
		FullName:   info.Subject.CommonName,
		Attributes: attrs,
	}
}

func options(info requestinfo.CertInfo) []tokens.Option {
	return []tokens.Option{tokens.WithExpiry(info.NotAfter)}
}
