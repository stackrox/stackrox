package basic

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/authproviders/idputil"
	"github.com/stackrox/rox/pkg/httputil"
)

const (
	// TypeName is the standard type name for basic auth provider.
	TypeName = "basic"
)

var (
	_ authproviders.BackendFactory = (*factory)(nil)
)

type factory struct {
	urlPathPrefix string
}

// NewFactory creates a new factory for Basic authprovider backends.
func NewFactory(urlPathPrefix string) authproviders.BackendFactory {
	urlPathPrefix = strings.TrimRight(urlPathPrefix, "/") + "/"
	return &factory{
		urlPathPrefix: urlPathPrefix,
	}
}

func (f *factory) CreateBackend(ctx context.Context, id string, _ []string, _ map[string]string, _ map[string]string) (authproviders.Backend, error) {
	providerURLPathPrefix := f.urlPathPrefix + id + "/"
	mgr := basicAuthManagerFromContext(ctx)
	if mgr == nil {
		return nil, errors.New("basic auth manager missing from context")
	}
	be, err := newBackend(providerURLPathPrefix, mgr)
	if err != nil {
		return nil, err
	}
	return be, nil
}

func (f *factory) ProcessHTTPRequest(_ http.ResponseWriter, r *http.Request) (string, string, error) {
	restPath := strings.TrimPrefix(r.URL.Path, f.urlPathPrefix)
	if len(restPath) == len(r.URL.Path) {
		return "", "", httputil.NewError(http.StatusNotFound, "Not Found")
	}
	if restPath == "" {
		return "", "", httputil.NewError(http.StatusForbidden, "Forbidden")
	}
	pathComponents := strings.SplitN(restPath, "/", 2)
	return pathComponents[0], r.URL.Query().Get(clientStateQueryParamName), nil
}

func (f *factory) RedactConfig(config map[string]string) map[string]string {
	return config
}

func (f *factory) MergeConfig(newCfg, _ map[string]string) map[string]string {
	return newCfg
}

func (f *factory) ResolveProviderAndClientState(state string) (string, string, error) {
	providerID, clientState := idputil.SplitState(state)
	if len(providerID) == 0 {
		return "", clientState, errors.New("empty state")
	}
	return providerID, clientState, nil
}

func (f *factory) GetSuggestedAttributes() []string {
	return []string{}
}
