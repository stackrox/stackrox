package userpki

import (
	"context"
	"crypto/x509"
	"net/http"
	"strings"

	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	// TypeName is the unique identifier for providers of this type
	TypeName = "userpki"
)

var _ authproviders.BackendFactory = (*factory)(nil)

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

func (f *factory) CreateBackend(ctx context.Context, id string, _ []string, config map[string]string, _ map[string]string) (authproviders.Backend, error) {
	pathPrefix := f.callbackURLPath + id + "/"
	be, err := newBackend(ctx, pathPrefix, f.callbacks, config)
	if err != nil {
		return nil, err
	}
	return be, nil
}

func (f *factory) ProcessHTTPRequest(_ http.ResponseWriter, r *http.Request) (providerID string, clientState string, err error) {
	if r.Method != http.MethodGet {
		return "", "", httputil.Errorf(http.StatusMethodNotAllowed, "invalid method %q, only GET requests are allowed", r.Method)
	}

	restURL := strings.TrimPrefix(r.URL.Path, f.callbackURLPath)
	if len(restURL) == len(r.URL.Path) {
		return "", "", utils.ShouldErr(httputil.Errorf(http.StatusNotFound, "invalid path %q, expected sub-path of %q", r.URL.Path, f.callbackURLPath))
	}

	if restURL == "" {
		return "", "", httputil.Errorf(http.StatusNotFound, "Not Found")
	}

	providerID, _ = stringutils.Split2(restURL, "/")
	return providerID, "", nil
}

func (f *factory) ResolveProviderAndClientState(state string) (providerID string, clientState string, err error) {
	return state, "", nil
}

func (f *factory) RedactConfig(config map[string]string) map[string]string {
	return config
}

func (f *factory) MergeConfig(newCfg, _ map[string]string) map[string]string {
	return newCfg
}

func (f *factory) GetSuggestedAttributes() []string {
	return []string{authproviders.UseridAttribute,
		authproviders.NameAttribute,
		authproviders.GroupsAttribute,
		authproviders.EmailAttribute}
}

// NewFactoryFactory is a method to return an authproviders.BackendFactory that contains a reference to the
// callback interface
func NewFactoryFactory(callbacks ProviderCallbacks) func(string) authproviders.BackendFactory {
	return func(callbackURLPath string) authproviders.BackendFactory {
		return &factory{callbackURLPath: callbackURLPath, callbacks: callbacks}
	}
}
