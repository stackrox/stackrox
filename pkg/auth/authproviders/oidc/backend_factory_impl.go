package oidc

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/authproviders/idputil"
	"github.com/stackrox/rox/pkg/cryptoutils"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/maputil"
)

const (
	// TypeName is the standard type name for OIDC auth providers.
	TypeName = "oidc"

	callbackRelativePath = "callback"

	nonceTTL     = 1 * time.Minute
	nonceByteLen = 20
)

var (
	log                              = logging.LoggerForModule()
	_   authproviders.BackendFactory = (*factory)(nil)
)

type factory struct {
	callbackURLPath     string
	noncePool           cryptoutils.NoncePool
	providerFactoryFunc providerFactoryFunc
	oauthExchange       exchangeFunc
}

// NewFactory creates a new factory for OIDC authprovider backends.
func NewFactory(urlPathPrefix string) authproviders.BackendFactory {
	urlPathPrefix = strings.TrimRight(urlPathPrefix, "/") + "/"
	return &factory{
		callbackURLPath:     fmt.Sprintf("%s%s", urlPathPrefix, callbackRelativePath),
		noncePool:           cryptoutils.NewThreadSafeNoncePool(cryptoutils.NewNonceGenerator(nonceByteLen, rand.Reader), nonceTTL),
		providerFactoryFunc: newWrappedOIDCProvider,
		oauthExchange:       oauthExchange,
	}
}

func (f *factory) CreateBackend(ctx context.Context, id string, uiEndpoints []string, config map[string]string, mappings map[string]string) (authproviders.Backend, error) {
	return newBackend(ctx, id, uiEndpoints, f.callbackURLPath, config, f.providerFactoryFunc, f.oauthExchange, f.noncePool, mappings)
}

func (f *factory) ProcessHTTPRequest(_ http.ResponseWriter, r *http.Request) (string, string, error) {
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
	if config[ClientSecretConfigKey] != "" {
		config = maputil.ShallowClone(config)
		config[ClientSecretConfigKey] = "*****"
	}
	return config
}

func (f *factory) MergeConfig(newCfg, oldCfg map[string]string) map[string]string {
	mergedCfg := maputil.ShallowClone(newCfg)
	// This handles the case where the client sends an "unchanged" client secret. In that case,
	// we will take the client secret from the stored config and put it into the merged config.
	// We only put secret into the merged config if the new config says it wants to use a client secret, AND the client
	// secret is not specified in the request.
	if mergedCfg[DontUseClientSecretConfigKey] == "false" && mergedCfg[ClientSecretConfigKey] == "" {
		mergedCfg[ClientSecretConfigKey] = oldCfg[ClientSecretConfigKey]
	}
	return mergedCfg
}

func (f *factory) GetSuggestedAttributes() []string {
	return []string{authproviders.UseridAttribute,
		authproviders.NameAttribute,
		authproviders.GroupsAttribute,
		authproviders.EmailAttribute}
}
