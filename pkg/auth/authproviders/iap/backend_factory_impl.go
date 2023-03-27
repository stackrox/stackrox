package iap

import (
	"context"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/stringutils"
)

const (
	// TypeName is the unique identifier for providers of this type
	TypeName = "iap"
	// AudienceConfigKey is the config key for the audience string for this provider
	AudienceConfigKey = "audience"
)

var _ authproviders.BackendFactory = (*factory)(nil)

type factory struct {
	callbackURL string
}

func (f *factory) CreateBackend(_ context.Context, id string, _ []string, config map[string]string, _ map[string]string) (authproviders.Backend, error) {
	audience := config[AudienceConfigKey]
	if audience == "" {
		return nil, errors.Errorf("parameter %q is required", audience)
	}
	loginURL := f.callbackURL + id
	return newBackend(audience, loginURL)
}

func (f *factory) ProcessHTTPRequest(_ http.ResponseWriter, r *http.Request) (string, string, error) {
	if r.Method != http.MethodGet {
		return "", "", httputil.Errorf(http.StatusMethodNotAllowed, "invalid method %q, only GET requests are allowed", r.Method)
	}

	if !strings.HasPrefix(r.URL.Path, f.callbackURL) {
		return "", "", httputil.NewError(http.StatusBadRequest, "invalid request url")
	}

	providerID, _ := stringutils.Split2(strings.TrimPrefix(r.URL.Path, f.callbackURL), "/")
	return providerID, "", nil
}

func (f *factory) ResolveProvider(_ string) (providerID string, err error) {
	return "", errors.New("unimplemented")
}

func (f *factory) RedactConfig(config map[string]string) map[string]string {
	return config
}

func (f *factory) MergeConfig(newCfg, _ map[string]string) map[string]string {
	return newCfg
}

func (f *factory) ResolveProviderAndClientState(state string) (providerID string, clientState string, err error) {
	return state, "", nil
}

func (f *factory) GetSuggestedAttributes() []string {
	return []string{authproviders.UseridAttribute,
		authproviders.EmailAttribute}
}

// NewFactory is a method to return an authproviders.BackendFactory that contains a reference to the
// callback interface
func NewFactory(callbackURLPath string) authproviders.BackendFactory {
	return &factory{
		callbackURL: strings.TrimRight(callbackURLPath, "/") + "/",
	}
}
