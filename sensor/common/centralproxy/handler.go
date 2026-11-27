package centralproxy

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/features"
	pkghttputil "github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/retryablehttp"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"google.golang.org/grpc"
	authenticationv1 "k8s.io/api/authentication/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	log            = logging.LoggerForModule()
	k8sClientQPS   = 50.0
	k8sClientBurst = 100

	_ common.Notifiable           = (*Handler)(nil)
	_ common.CentralGRPCConnAware = (*Handler)(nil)
)

// proxyErrorHandler is the error handler for the reverse proxy.
// It returns 503 for service unavailable errors and 500 for other errors.
func proxyErrorHandler(w http.ResponseWriter, _ *http.Request, err error) {
	log.Errorf("Proxy error: %v", err)
	if errors.Is(err, errServiceUnavailable) {
		http.Error(w, fmt.Sprintf("proxy temporarily unavailable: %v", err), http.StatusServiceUnavailable)
		return
	}
	http.Error(w, fmt.Sprintf("failed to contact central: %v", err), http.StatusInternalServerError)
}

// Handler handles HTTP proxy requests to Central.
type Handler struct {
	centralReachable atomic.Bool
	authorizer       *k8sAuthorizer
	policy           *ClientPolicy
	transport        *scopedTokenTransport
	proxy            *httputil.ReverseProxy
}

// NewProxyHandler creates a new proxy handler that forwards requests to Central.
func NewProxyHandler(centralEndpoint string, centralCertificates []*x509.Certificate, clusterIDGetter clusterIDGetter) (*Handler, error) {
	centralBaseURL, err := url.Parse(
		urlfmt.FormatURL(centralEndpoint, urlfmt.HTTPS, urlfmt.NoTrailingSlash),
	)
	if err != nil {
		return nil, errors.Wrap(err, "parsing endpoint")
	}

	baseTransport, err := createBaseTransport(centralBaseURL, centralCertificates)
	if err != nil {
		return nil, errors.Wrap(err, "creating base transport")
	}

	transport := newScopedTokenTransport(baseTransport, clusterIDGetter)

	proxy := &httputil.ReverseProxy{
		Transport:    transport,
		Rewrite:      func(r *httputil.ProxyRequest) { r.SetURL(centralBaseURL) },
		ErrorHandler: proxyErrorHandler,
	}

	restConfig, err := k8sutil.GetK8sInClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "getting in-cluster config")
	}
	// Set QPS and Burst to avoid client-side throttling.
	// The default k8s client-go values (QPS=5, Burst=10) are quite conservative
	// and can cause "client-side throttling, not priority and fairness" warnings.
	restConfig.QPS = float32(k8sClientQPS)
	restConfig.Burst = k8sClientBurst
	retryablehttp.ConfigureRESTConfig(restConfig)

	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "creating kubernetes client")
	}
	policy := NewClientPolicy(k8sClient)
	if err := policy.EnsureDefaults(context.Background()); err != nil {
		return nil, errors.Wrap(err, "initializing proxy client policy")
	}

	return &Handler{
		authorizer: newK8sAuthorizer(k8sClient),
		policy:     policy,
		transport:  transport,
		proxy:      proxy,
	}, nil
}

// SetCentralGRPCClient implements common.CentralGRPCConnAware.
// It sets the gRPC connection used by the token provider to request tokens from Central.
func (h *Handler) SetCentralGRPCClient(cc grpc.ClientConnInterface) {
	h.transport.SetClient(cc)
}

// Notify reacts to sensor going into online/offline mode.
func (h *Handler) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e, "Central proxy handler"))
	switch e {
	case common.SensorComponentEventCentralReachable:
		h.centralReachable.Store(true)
	case common.SensorComponentEventOfflineMode:
		h.centralReachable.Store(false)
	}
}

// validateRequest validates the incoming request and returns an error if validation fails.
func (h *Handler) validateRequest(request *http.Request) error {
	if !h.centralReachable.Load() {
		return pkghttputil.NewError(http.StatusServiceUnavailable, "central not reachable")
	}
	return nil
}

func validateClientRequest(grant *ClientPolicyGrant, request *http.Request) error {
	if grant.isConsole() && !features.OCPConsoleIntegration.Enabled() {
		return pkghttputil.NewError(http.StatusForbidden, "console proxy is disabled")
	}
	return nil
}

// checkInternalTokenAPISupport checks if Central supports the internal token API capability.
// The proxy requires this capability to function; all requests are rejected if unsupported.
func checkInternalTokenAPISupport() error {
	if !centralcaps.Has(centralsensor.InternalTokenAPISupported) {
		return pkghttputil.NewError(http.StatusNotImplemented,
			"proxy to Central is not available; Central does not support the internal token API required by this proxy")
	}
	return nil
}

// ServeHTTP handles incoming HTTP requests and proxies them to Central.
func (h *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	start := time.Now()
	result := requestResultSuccess
	defer func() {
		observeProxyRequest(result, time.Since(start))
	}()

	if err := checkInternalTokenAPISupport(); err != nil {
		result = requestResultNotImplemented
		http.Error(writer, err.Error(), pkghttputil.StatusFromError(err))
		return
	}

	if err := h.validateRequest(request); err != nil {
		result = requestResultValidationError
		http.Error(writer, err.Error(), pkghttputil.StatusFromError(err))
		return
	}

	if h.policy == nil {
		result = requestResultConfigError
		http.Error(writer, "proxy policy not configured", http.StatusInternalServerError)
		return
	}
	var userInfo *authenticationv1.UserInfo
	grant, err := h.policy.authorizeRequest(request.Context(), request, func() (*authenticationv1.UserInfo, error) {
		if h.authorizer == nil {
			result = requestResultConfigError
			return nil, pkghttputil.NewError(http.StatusInternalServerError, "authorizer not configured")
		}
		var err error
		userInfo, err = h.authorizer.authenticate(request.Context(), request)
		if err != nil {
			result = requestResultAuthnError
		}
		return userInfo, err
	})
	if err != nil {
		if result == requestResultSuccess {
			result = requestResultAuthzError
			if pkghttputil.StatusFromError(err) == http.StatusUnauthorized {
				result = requestResultAuthnError
			}
		}
		http.Error(writer, err.Error(), pkghttputil.StatusFromError(err))
		return
	}
	if err := validateClientRequest(grant, request); err != nil {
		result = requestResultValidationError
		http.Error(writer, err.Error(), pkghttputil.StatusFromError(err))
		return
	}
	permissionSetSupported := centralcaps.Has(centralsensor.CentralProxyPermissionSetTokens)
	legacyConsole := grant.isConsole() && grant.AuthType == clientAuthKubernetesBearer
	if !permissionSetSupported && !legacyConsole {
		result = requestResultNotImplemented
		http.Error(writer, "Central does not support this proxy client", http.StatusNotImplemented)
		return
	}
	profile := &proxyTokenProfile{
		identity:       grant.AuthType + ":" + grant.Subject,
		permissionSets: grant.AllowedPermissionSets,
		legacyConsole:  !permissionSetSupported && legacyConsole,
	}
	if userInfo == nil {
		profile.clusterScope = FullClusterAccessScope
	} else {
		profile.clusterScope = request.Header.Get(stackroxNamespaceHeader)
		if profile.clusterScope == "" && !grant.isConsole() {
			result = requestResultAuthzError
			http.Error(writer, "namespace scope is required for custom bearer clients", http.StatusForbidden)
			return
		}
		if err := h.authorizer.authorize(request.Context(), userInfo, request); err != nil {
			result = requestResultAuthzError
			http.Error(writer, err.Error(), pkghttputil.StatusFromError(err))
			return
		}
	}
	request = request.WithContext(withProxyTokenProfile(request.Context(), profile))

	tracker := pkghttputil.NewStatusTrackingWriter(writer)
	h.proxy.ServeHTTP(tracker, request)

	if statusCode := tracker.GetStatusCode(); statusCode != nil && *statusCode >= http.StatusBadRequest {
		result = requestResultProxyError
	}
}
