package centralproxy

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/centralproxy"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	pkghttputil "github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/retryablehttp"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/centralproxy/allowedpaths"
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
	clusterIDGetter  clusterIDGetter
	authorizer       *k8sAuthorizer
	policy           *ClientPolicy
	transport        *scopedTokenTransport
	proxy            *httputil.ReverseProxy
	metricsEnabled   bool
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
		clusterIDGetter: clusterIDGetter,
		authorizer:      newK8sAuthorizer(k8sClient),
		policy:          policy,
		transport:       transport,
		proxy:           proxy,
		metricsEnabled:  env.SecuredClusterMetricsEnabled.BooleanSetting(),
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
	if !isMetricsPath(request) && !features.OCPConsoleIntegration.Enabled() {
		return pkghttputil.NewError(http.StatusForbidden, "console proxy is disabled")
	}
	if isMetricsPath(request) {
		if request.URL.EscapedPath() != "/metrics" || request.Method != http.MethodGet {
			return pkghttputil.NewError(http.StatusForbidden, "invalid metrics scrape path or method")
		}
		if !h.metricsEnabled || !centralcaps.Has(centralsensor.CentralProxyPathFiltering) ||
			!allowedpaths.IsExplicitlyAllowed("/metrics") {
			return pkghttputil.NewError(http.StatusForbidden, "secured-cluster metrics proxy unavailable")
		}
	}
	// Allow GET, POST, OPTIONS (for CORS preflight), and HEAD.
	switch request.Method {
	case http.MethodGet, http.MethodPost, http.MethodOptions, http.MethodHead:
		// allowed
	default:
		return pkghttputil.Errorf(http.StatusMethodNotAllowed, "method %s not allowed", request.Method)
	}

	if !h.centralReachable.Load() {
		return pkghttputil.NewError(http.StatusServiceUnavailable, "central not reachable")
	}

	normalizedPath := request.URL.EscapedPath()
	if normalizedPath == "" {
		normalizedPath = "/"
	}
	if !isProxyPathAllowed(normalizedPath) {
		return pkghttputil.Errorf(http.StatusForbidden,
			"path %q is not allowed by the proxy allow-list", request.URL.Path)
	}

	return nil
}

func isProxyPathAllowed(path string) bool {
	if !centralcaps.Has(centralsensor.CentralProxyPathFiltering) {
		// Older Centrals cannot advertise paths and issue only hardcoded read-only console tokens.
		return true
	}
	return allowedpaths.IsAllowed(path)
}

func isMetricsPath(request *http.Request) bool {
	decoded := request.URL.Path
	for range 16 {
		if path.Clean(decoded) == "/metrics" {
			return true
		}
		next, err := url.PathUnescape(decoded)
		if err != nil || next == decoded {
			return false
		}
		decoded = next
	}
	return strings.Contains(decoded, "%")
}

func authenticateMetricsScrape(request *http.Request) error {
	if request.TLS == nil || len(request.TLS.VerifiedChains) == 0 ||
		len(request.TLS.VerifiedChains[0]) == 0 ||
		request.TLS.VerifiedChains[0][0].Subject.CommonName != env.SecureMetricsClientCertCN.Setting() {
		return pkghttputil.NewError(http.StatusUnauthorized, "verified Prometheus client certificate required")
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
	if h.policy != nil {
		h.servePolicyRequest(writer, request, &result)
		return
	}

	if err := h.validateRequest(request); err != nil {
		result = requestResultValidationError
		http.Error(writer, err.Error(), pkghttputil.StatusFromError(err))
		return
	}
	if isMetricsPath(request) {
		if err := authenticateMetricsScrape(request); err != nil {
			result = requestResultAuthnError
			http.Error(writer, err.Error(), pkghttputil.StatusFromError(err))
			return
		}
		request = request.WithContext(withMetricsScrape(request.Context()))
		tracker := pkghttputil.NewStatusTrackingWriter(writer)
		h.proxy.ServeHTTP(tracker, request)
		if statusCode := tracker.GetStatusCode(); statusCode != nil && *statusCode >= http.StatusBadRequest {
			result = requestResultProxyError
		}
		return
	}

	if h.authorizer == nil {
		result = requestResultConfigError
		log.Error("Authorizer is nil - this indicates a misconfiguration in the central proxy handler")
		http.Error(writer, "authorizer not configured", http.StatusInternalServerError)
		return
	}

	userInfo, err := h.authorizer.authenticate(request.Context(), request)
	if err != nil {
		result = requestResultAuthnError
		http.Error(writer, err.Error(), pkghttputil.StatusFromError(err))
		return
	}

	if err := h.authorizer.authorize(request.Context(), userInfo, request); err != nil {
		result = requestResultAuthzError
		http.Error(writer, err.Error(), pkghttputil.StatusFromError(err))
		return
	}

	tracker := pkghttputil.NewStatusTrackingWriter(writer)
	h.proxy.ServeHTTP(tracker, request)

	if statusCode := tracker.GetStatusCode(); statusCode != nil && *statusCode >= http.StatusBadRequest {
		result = requestResultProxyError
	}
}

func (h *Handler) servePolicyRequest(writer http.ResponseWriter, request *http.Request, result *string) {
	if !h.centralReachable.Load() {
		*result = requestResultValidationError
		http.Error(writer, "Central is not reachable", http.StatusServiceUnavailable)
		return
	}
	if isMetricsPath(request) && !h.metricsEnabled && !env.CentralProxyEnabled.BooleanSetting() {
		*result = requestResultValidationError
		http.Error(writer, "secured-cluster metrics proxy unavailable", http.StatusForbidden)
		return
	}
	if h.authorizer == nil {
		*result = requestResultConfigError
		http.Error(writer, "proxy authorizer not configured", http.StatusInternalServerError)
		return
	}
	var userInfo *authenticationv1.UserInfo
	if request.TLS == nil || len(request.TLS.PeerCertificates) == 0 {
		var err error
		userInfo, err = h.authorizer.authenticate(request.Context(), request)
		if err != nil {
			*result = requestResultAuthnError
			http.Error(writer, err.Error(), pkghttputil.StatusFromError(err))
			return
		}
	}
	grant, err := h.policy.Authorize(request.Context(), request, userInfo)
	if err != nil {
		*result = requestResultAuthzError
		http.Error(writer, err.Error(), pkghttputil.StatusFromError(err))
		return
	}
	if grant.PermissionSetID == centralproxy.ConsoleProxyPermissionSetID && !features.OCPConsoleIntegration.Enabled() {
		*result = requestResultValidationError
		http.Error(writer, "console proxy is disabled", http.StatusForbidden)
		return
	}
	if !centralcaps.Has(centralsensor.CentralProxyCustomReadOnlyTokens) && !legacyProxyGrantAllowed(grant, request) {
		*result = requestResultNotImplemented
		http.Error(writer, "Central does not support this proxy client profile", http.StatusNotImplemented)
		return
	}
	if userInfo != nil {
		if err := h.authorizer.authorize(request.Context(), userInfo, request); err != nil {
			*result = requestResultAuthzError
			http.Error(writer, err.Error(), pkghttputil.StatusFromError(err))
			return
		}
	}
	profile := &proxyTokenProfile{
		identity:        grant.AuthType + ":" + grant.Subject,
		permissionSetID: grant.PermissionSetID,
		httpScopes:      grant.HTTPRequestScopes,
		legacyConsole:   grant.PermissionSetID == centralproxy.ConsoleProxyPermissionSetID,
	}
	if userInfo == nil || grant.PermissionSetID == centralproxy.MetricsPermissionSetID {
		profile.clusterScope = FullClusterAccessScope
	} else {
		profile.clusterScope = request.Header.Get(stackroxNamespaceHeader)
		if profile.clusterScope == "" && !profile.legacyConsole {
			*result = requestResultAuthzError
			http.Error(writer, "namespace scope is required for custom bearer clients", http.StatusForbidden)
			return
		}
	}
	request = request.WithContext(withProxyTokenProfile(request.Context(), profile))
	tracker := pkghttputil.NewStatusTrackingWriter(writer)
	h.proxy.ServeHTTP(tracker, request)
	if statusCode := tracker.GetStatusCode(); statusCode != nil && *statusCode >= http.StatusBadRequest {
		*result = requestResultProxyError
	}
}

func legacyProxyGrantAllowed(grant *ClientPolicyGrant, request *http.Request) bool {
	var allowed tokens.HTTPRequestScopes
	switch grant.PermissionSetID {
	case centralproxy.ConsoleProxyPermissionSetID:
		allowed = centralproxy.ConsoleHTTPScopes
	case centralproxy.MetricsPermissionSetID:
		if !centralcaps.Has(centralsensor.CentralProxyPathFiltering) || !allowedpaths.IsExplicitlyAllowed("/metrics") {
			return false
		}
		allowed = centralproxy.MetricsHTTPScopes
	default:
		return false
	}
	if centralcaps.Has(centralsensor.CentralProxyPathFiltering) && !allowedpaths.IsAllowed(request.URL.EscapedPath()) {
		return false
	}
	for _, scope := range grant.HTTPRequestScopes {
		methods, found := allowed[scope.GetPath()]
		if !found {
			return false
		}
		for _, method := range scope.GetMethods() {
			if !slices.Contains(methods, method) {
				return false
			}
		}
	}
	return true
}
