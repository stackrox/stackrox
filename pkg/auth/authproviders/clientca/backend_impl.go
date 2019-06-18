package clientca

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/http"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// ConfigKeys is the map key in the provider configuration
	ConfigKeys = "keys"
)

var (
	log = logging.LoggerForModule()
)

func newBackend(ctx context.Context, id string, callbacks ProviderCallbacks, config map[string]string) (authproviders.Backend, map[string]string, error) {
	pem := config[ConfigKeys]
	if pem == "" {
		return nil, nil, fmt.Errorf("Parameter %q is required", ConfigKeys)
	}
	certs, err := helpers.ParseCertificatesPEM([]byte(pem))
	if err != nil {
		return nil, nil, err
	}
	return &backendImpl{
		callbacks: callbacks,
		certs:     certs,
	}, config, nil
}

type backendImpl struct {
	callbacks ProviderCallbacks
	certs     []*x509.Certificate
}

func (b *backendImpl) OnEnable(provider authproviders.Provider) {
	log.Debugf("Provider %q enabled", provider.ID())
	b.callbacks.RegisterAuthProvider(provider, b.certs)
}

func (b *backendImpl) OnDisable(provider authproviders.Provider) {
	log.Debugf("Provider %q disabled", provider.ID())
	b.callbacks.UnregisterAuthProvider(provider)
}

func (b *backendImpl) LoginURL(clientState string, ri *requestinfo.RequestInfo) string {
	return ""
}

func (b *backendImpl) RefreshURL() string {
	return ""
}

func (b *backendImpl) ProcessHTTPRequest(w http.ResponseWriter, r *http.Request) (*tokens.ExternalUserClaim, []tokens.Option, string, error) {
	return nil, nil, "", status.Errorf(codes.Unimplemented, "ProcessHTTPRequest not implemented for provider type %q", TypeName)
}

func (b *backendImpl) ExchangeToken(ctx context.Context, externalToken, state string) (*tokens.ExternalUserClaim, []tokens.Option, string, error) {
	return nil, nil, "", status.Errorf(codes.Unimplemented, "ExchangeToken not implemented for provider type %q", TypeName)
}
