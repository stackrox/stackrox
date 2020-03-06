package oidc

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/authproviders/idputil"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/maputil"
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

func (f *factory) CreateBackend(ctx context.Context, id string, uiEndpoints []string, config map[string]string) (authproviders.Backend, error) {
	be, err := newBackend(ctx, id, uiEndpoints, f.callbackURLPath, config)
	if err != nil {
		return nil, err
	}
	return be, nil
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
	providerID, _ := idputil.SplitState(state)
	if providerID == "" {
		return "", httputil.NewError(http.StatusBadRequest, "malformed state")
	}

	return providerID, nil
}

func (f *factory) RedactConfig(config map[string]string) map[string]string {
	if config[clientSecretConfigKey] != "" {
		config = maputil.CloneStringStringMap(config)
		config[clientSecretConfigKey] = "*****"
	}
	return config
}

func (f *factory) MergeConfig(newCfg, oldCfg map[string]string) map[string]string {
	mergedCfg := maputil.CloneStringStringMap(newCfg)
	// This handles the case where the client sends an "unchanged" client secret. In that case,
	// we will take the client secret from the stored config and put it into the merged config.
	// We only put secret into the merged config if the new config says it wants to use a client secret, AND the client
	// secret is not specified in the request.
	if mergedCfg[dontUseClientSecretConfigKey] == "false" && mergedCfg[clientSecretConfigKey] == "" {
		mergedCfg[clientSecretConfigKey] = oldCfg[clientSecretConfigKey]
	}
	return mergedCfg
}
