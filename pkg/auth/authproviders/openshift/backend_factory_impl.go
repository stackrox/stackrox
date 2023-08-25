package openshift

import (
	"context"
	"net/http"
	"strings"

	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/authproviders/idputil"
	"github.com/stackrox/rox/pkg/httputil"
)

const (
	// TypeName is the standard type name for OpenShift auth providers.
	TypeName = "openshift"

	callbackRelativePath = "callback"
)

type factory struct {
	callbackURLPath string
}

var _ authproviders.BackendFactory = (*factory)(nil)

// NewFactory creates a new factory for OpenShift oauth authprovider backends.
func NewFactory(urlPathPrefix string) authproviders.BackendFactory {
	urlPathPrefix = strings.TrimRight(urlPathPrefix, "/") + "/"
	return &factory{
		callbackURLPath: urlPathPrefix + callbackRelativePath,
	}
}

func (f *factory) CreateBackend(_ context.Context, id string, _ []string, config map[string]string, _ map[string]string) (authproviders.Backend, error) {
	return newBackend(id, f.callbackURLPath, config)
}

func (f *factory) ProcessHTTPRequest(_ http.ResponseWriter, r *http.Request) (providerID string, clientState string, err error) {
	if r.URL.Path != f.callbackURLPath {
		return "", "", httputil.NewError(http.StatusNotFound, "Not Found")
	}

	values, err := authproviders.ExtractURLValuesFromRequest(r)
	if err != nil {
		return "", "", err
	}

	return f.ResolveProviderAndClientState(values.Get("state"))
}

func (f *factory) ResolveProviderAndClientState(state string) (string, string, error) {
	providerID, clientState := idputil.SplitState(state)
	if providerID == "" {
		return "", clientState, httputil.NewError(http.StatusBadRequest, "malformed state")
	}

	return providerID, clientState, nil
}

func (f *factory) RedactConfig(config map[string]string) map[string]string {
	return config
}

func (f *factory) MergeConfig(newConfig, _ map[string]string) map[string]string {
	return newConfig
}

func (f *factory) GetSuggestedAttributes() []string {
	return []string{authproviders.UseridAttribute,
		authproviders.NameAttribute,
		authproviders.GroupsAttribute}
}
