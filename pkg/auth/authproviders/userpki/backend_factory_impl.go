package userpki

import (
	"context"
	"crypto/x509"
	"net/http"

	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
)

const (
	// TypeName is the unique identifier for providers of this type
	TypeName = "userpki"
)

type factory struct {
	callbackURLPath string
	callbacks       ProviderCallbacks
}

// ProviderCallbacks is an interface for ClientCAManager to implement
type ProviderCallbacks interface {
	RegisterAuthProvider(provider authproviders.Provider, certs []*x509.Certificate)
	UnregisterAuthProvider(provider authproviders.Provider)
	GetProviderForFingerprint(fingerprint string) authproviders.Provider
}

func (f *factory) CreateBackend(ctx context.Context, id string, uiEndpoints []string, config map[string]string) (authproviders.Backend, error) {
	pathPrefix := f.callbackURLPath + id + "/"
	be, err := newBackend(ctx, pathPrefix, f.callbacks, config)
	if err != nil {
		return nil, err
	}
	return be, nil
}

func (f *factory) ProcessHTTPRequest(w http.ResponseWriter, r *http.Request) (providerID string, err error) {
	ri := requestinfo.FromContext(r.Context())
	if len(ri.VerifiedChains) != 1 {
		return "", errNoCertificate
	}
	for _, cert := range ri.VerifiedChains[0] {
		if prov := f.callbacks.GetProviderForFingerprint(cert.CertFingerprint); prov != nil {
			return prov.ID(), nil
		}
	}

	return "", errInvalidCertificate
}

func (f *factory) ResolveProvider(state string) (providerID string, err error) {
	return state, nil
}

func (f *factory) RedactConfig(config map[string]string) map[string]string {
	return config
}

func (f *factory) MergeConfig(newCfg, oldCfg map[string]string) map[string]string {
	return newCfg
}

// NewFactoryFactory is a method to return an authproviders.BackendFactory that contains a reference to the
// callback interface
func NewFactoryFactory(callbacks ProviderCallbacks) func(string) authproviders.BackendFactory {
	return func(callbackURLPath string) authproviders.BackendFactory {
		return &factory{callbackURLPath: callbackURLPath, callbacks: callbacks}
	}
}
