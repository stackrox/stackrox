package clientca

import (
	"context"
	"crypto/x509"
	"net/http"

	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/httputil"
)

const (
	// TypeName is the unique identifier for providers of this type
	TypeName = "clientca"
)

type factory struct {
	callbacks ProviderCallbacks
}

// ProviderCallbacks is an interface for ClientCAManager to implement
type ProviderCallbacks interface {
	RegisterAuthProvider(provider authproviders.Provider, certs []*x509.Certificate)
	UnregisterAuthProvider(provider authproviders.Provider)
}

func (f *factory) CreateBackend(ctx context.Context, id string, uiEndpoints []string, config map[string]string) (authproviders.Backend, map[string]string, error) {
	return newBackend(ctx, id, f.callbacks, config)

}

func (f *factory) ProcessHTTPRequest(w http.ResponseWriter, r *http.Request) (providerID string, err error) {
	return "", httputil.NewError(http.StatusInternalServerError, "received invalid request")
}

func (f *factory) ResolveProvider(state string) (providerID string, err error) {
	return state, nil
}

// NewFactoryFactory is a method to return an authproviders.BackendFactory that contains a reference to the
// callback interface
func NewFactoryFactory(callbacks ProviderCallbacks) func(string) authproviders.BackendFactory {
	return func(string) authproviders.BackendFactory {
		return &factory{callbacks: callbacks}
	}
}
