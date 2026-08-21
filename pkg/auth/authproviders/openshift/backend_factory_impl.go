package openshift

import (
	"context"
	"net/http"
	"strings"

	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/authproviders/idputil"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/maputil"
)

const (
	// TypeName is the standard type name for OpenShift auth providers.
	TypeName = "openshift"

	// TypeNameWithACMAccessControlDelegation is the type name for OpenShift auth providers
	// with role resolution from ACM.
	TypeNameWithACMAccessControlDelegation = "openshift-with-acm-roles"

	callbackRelativePath = "callback"
)

type newBackendFunc func(id string, callbackURL string, _ map[string]string) (*backend, error)

type factory struct {
	callbackURLPath string
	newBackend      newBackendFunc
}

var _ authproviders.BackendFactory = (*factory)(nil)

// NewFactory creates a new factory for OpenShift oauth authprovider backends.
func NewFactory(urlPathPrefix string) authproviders.BackendFactory {
	urlPathPrefix = strings.TrimRight(urlPathPrefix, "/") + "/"
	return &factory{
		callbackURLPath: urlPathPrefix + callbackRelativePath,
		newBackend:      newBackend,
	}
}

// NewFactoryWithACMAccessControlDelegation creates a new factory for OpenShift oauth authprovider backends.
func NewFactoryWithACMAccessControlDelegation(urlPathPrefix string) authproviders.BackendFactory {
	urlPathPrefix = strings.TrimRight(urlPathPrefix, "/") + "/"
	return &factory{
		callbackURLPath: urlPathPrefix + callbackRelativePath,
		newBackend:      newBackendWithACMAccessControlDelegation,
	}
}

func (f *factory) CreateBackend(_ context.Context, id string, _ []string, config map[string]string, _ map[string]string) (authproviders.Backend, error) {
	return f.newBackend(id, f.callbackURLPath, config)
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
	if features.ACMAccessControlDelegation.Enabled() {
		if config[ClientSecretConfigKey] != "" {
			config = maputil.ShallowClone(config)
			config[ClientSecretConfigKey] = "*****"
		}
	}
	return config
}

func (f *factory) MergeConfig(newConfig, oldConfig map[string]string) map[string]string {
	if features.ACMAccessControlDelegation.Enabled() {
		mergedCfg := maputil.ShallowClone(oldConfig)
		if newConfig[ClientNameConfigKey] != "" {
			mergedCfg[ClientNameConfigKey] = newConfig[ClientNameConfigKey]
		}
		if newConfig[ClientSecretConfigKey] != "" {
			mergedCfg[ClientSecretConfigKey] = newConfig[ClientSecretConfigKey]
		}
		return mergedCfg
	}
	return newConfig
}

func (f *factory) GetSuggestedAttributes() []string {
	return []string{authproviders.UseridAttribute,
		authproviders.NameAttribute,
		authproviders.GroupsAttribute}
}
