package centralproxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"

	apiV1 "github.com/stackrox/rox/generated/api/v1"
	centralv1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	pkghttputil "github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/mtls/clientca"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/centralproxy/allowedpaths"
	authenticationv1 "k8s.io/api/authentication/v1"
	"k8s.io/client-go/kubernetes"
)

// fixedClientPolicy supports the built-in console and Prometheus integrations.
type fixedClientPolicy struct {
	client kubernetes.Interface
}

func (p *fixedClientPolicy) authorizeRequest(_ context.Context, request *http.Request,
	authenticate func() (*authenticationv1.UserInfo, error),
) (*ClientPolicyGrant, error) {
	if err := p.validateRequest(request); err != nil {
		return nil, err
	}
	if isMetricsPath(request) {
		if err := authenticateMetricsScrape(request); err != nil {
			return nil, err
		}
		return &ClientPolicyGrant{
			AuthType: clientAuthMTLS, Subject: env.SecureMetricsClientCertCN.Setting(),
			PermissionSets: metricsPermissionSets(),
		}, nil
	}
	user, err := authenticate()
	if err != nil {
		return nil, err
	}
	return &ClientPolicyGrant{
		AuthType: clientAuthKubernetesBearer, Subject: user.Username,
		PermissionSets: consolePermissionSets(),
	}, nil
}

func (p *fixedClientPolicy) validateRequest(request *http.Request) error {
	if isMetricsPath(request) {
		if request.URL.EscapedPath() != "/metrics" || request.Method != http.MethodGet {
			return pkghttputil.NewError(http.StatusForbidden, "invalid metrics scrape path or method")
		}
		if !centralcaps.Has(centralsensor.CentralProxyPerAccessTokens) {
			return pkghttputil.NewError(http.StatusForbidden, "secured-cluster metrics proxy unavailable")
		}
		return nil
	}
	if !features.OCPConsoleIntegration.Enabled() {
		return pkghttputil.NewError(http.StatusForbidden, "console proxy is disabled")
	}
	switch request.Method {
	case http.MethodGet, http.MethodPost, http.MethodOptions, http.MethodHead:
	default:
		return pkghttputil.Errorf(http.StatusMethodNotAllowed, "method %s not allowed", request.Method)
	}
	normalizedPath := request.URL.EscapedPath()
	if normalizedPath == "" {
		normalizedPath = "/"
	}
	if centralcaps.Has(centralsensor.CentralProxyPathFiltering) && !allowedpaths.IsAllowed(normalizedPath) {
		return pkghttputil.Errorf(http.StatusForbidden,
			"path %q is not allowed by the proxy allow-list", request.URL.Path)
	}
	return nil
}

func consolePermissionSets() []*centralv1.ScopedPermissionSet {
	return []*centralv1.ScopedPermissionSet{{
		Selector:          &centralv1.ScopedPermissionSet_Id{Id: accesscontrol.DefaultPermissionSetIDs[accesscontrol.K8sCentralProxyUser]},
		HttpRequestScopes: consoleRequestScopes(),
	}}
}

func metricsPermissionSets() []*centralv1.ScopedPermissionSet {
	return []*centralv1.ScopedPermissionSet{{
		Selector:          &centralv1.ScopedPermissionSet_Name{Name: accesscontrol.OpenShiftPrometheus},
		HttpRequestScopes: []*apiV1.HTTPRequestScope{{Path: "/metrics", Methods: []string{http.MethodGet}}},
	}}
}

func authenticateMetricsScrape(request *http.Request) error {
	if request.TLS == nil || len(request.TLS.VerifiedChains) == 0 ||
		len(request.TLS.VerifiedChains[0]) == 0 || request.TLS.VerifiedChains[0][0] == nil ||
		request.TLS.VerifiedChains[0][0].Subject.CommonName != env.SecureMetricsClientCertCN.Setting() {
		return pkghttputil.NewError(http.StatusUnauthorized, "verified Prometheus client certificate required")
	}
	return nil
}

func (p *fixedClientPolicy) configureTLS(config *tls.Config, updateClientCA func([]*x509.Certificate)) {
	if !env.SecuredClusterMetricsEnabled.BooleanSetting() {
		return
	}
	config.ClientAuth = tls.VerifyClientCertIfGiven
	watcher := clientca.NewWatcher(p.client, updateClientCA)
	watcher.Watch(context.Background(), env.SecureMetricsClientCANamespace.Setting(),
		env.SecureMetricsClientCAConfigMap.Setting())
}
