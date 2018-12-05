package oidc

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
)

const (
	// TypeName is the standard type name for OIDC auth providers.
	TypeName = "oidc"

	callbackRelativePath = "callback"
)

var (
	log = logging.LoggerForModule()
)

type factory struct {
	callbackURLPath string
}

// NewFactory creates a new factory for OIDC authprovider backends.
func NewFactory(urlPathPrefix string) authproviders.BackendFactory {
	urlPathPrefix = strings.TrimRight(urlPathPrefix, "/") + "/"
	return &factory{
		callbackURLPath: fmt.Sprintf("%s%s", urlPathPrefix, callbackRelativePath),
	}
}

func (f *factory) CreateAuthProviderBackend(ctx context.Context, id string, uiEndpoints []string, config map[string]string) (authproviders.AuthProviderBackend, map[string]string, error) {
	return newProvider(ctx, id, uiEndpoints, f.callbackURLPath, config)
}

func (f *factory) ProcessHTTPRequest(w http.ResponseWriter, r *http.Request) (string, error) {
	if r.URL.Path != f.callbackURLPath {
		return "", httputil.NewError(http.StatusNotFound, "Not Found")
	}

	if r.Method != http.MethodPost {
		return "", httputil.Errorf(http.StatusMethodNotAllowed, "method %s is not supported for this URL", r.Method)
	}

	if err := r.ParseForm(); err != nil {
		return "", httputil.Errorf(http.StatusBadRequest, "could not parse form data: %v", err)
	}

	return f.ResolveProvider(r.FormValue("state"))
}

func (f *factory) ResolveProvider(state string) (string, error) {
	providerID, _ := splitState(state)
	if providerID == "" {
		return "", httputil.NewError(http.StatusBadRequest, "malformed state")
	}

	return providerID, nil
}
