package saml

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/httputil"
)

const (
	acsRelativePath = "acs"

	// TypeName is the standard type name for SAML auth providers
	TypeName = "saml"
)

type factory struct {
	urlPathPrefix string
}

// NewFactory creates a new SAML auth provider factory.
func NewFactory(urlPathPrefix string) authproviders.BackendFactory {
	urlPathPrefix = strings.TrimRight(urlPathPrefix, "/") + "/"
	f := &factory{
		urlPathPrefix: urlPathPrefix,
	}

	return f
}

func (f *factory) CreateAuthProviderBackend(ctx context.Context, id string, uiEndpoints []string, config map[string]string) (authproviders.Backend, map[string]string, error) {
	return newProvider(ctx, f.urlPathPrefix+acsRelativePath, id, uiEndpoints, config)
}

func (f *factory) processACSRequest(r *http.Request) (string, error) {
	if r.Method != http.MethodPost {
		return "", httputil.NewError(http.StatusMethodNotAllowed, "only POST requests are allowed to this URL")
	}
	if err := r.ParseForm(); err != nil {
		return "", httputil.Errorf(http.StatusBadRequest, "could not parse form data: %v", err)
	}
	state := r.FormValue("RelayState")
	providerID, _ := splitState(state)
	if providerID == "" {
		return "", httputil.NewError(http.StatusBadRequest, "malformed RelayState")
	}
	return providerID, nil
}

func (f *factory) ProcessHTTPRequest(w http.ResponseWriter, r *http.Request) (string, error) {
	if !strings.HasPrefix(r.URL.Path, f.urlPathPrefix) {
		return "", httputil.NewError(http.StatusInternalServerError, "received invalid request")
	}

	relativePath := r.URL.Path[len(f.urlPathPrefix):]
	if relativePath == acsRelativePath {
		return f.processACSRequest(r)
	}

	return "", httputil.NewError(http.StatusNotFound, "Not Found")
}

func (f *factory) ResolveProvider(state string) (string, error) {
	providerID, _ := splitState(state)
	if providerID == "" {
		return "", fmt.Errorf("malformed state %q", state)
	}
	return providerID, nil
}
