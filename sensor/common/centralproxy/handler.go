package centralproxy

import (
	"crypto/x509"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
	"sync/atomic"

	"github.com/pkg/errors"
	pkghttputil "github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/retryablehttp"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/sensor/common"
	"k8s.io/client-go/kubernetes"
)

var (
	log = logging.LoggerForModule()

	_ common.Notifiable = (*Handler)(nil)

	// authzSkipPathPrefixes contains endpoint path prefixes that skip Central authorization checks
	// but still require authentication. Matching enforces segment boundaries: exact match or path
	// starting with prefix + "/".
	authzSkipPathPrefixes = []string{
		"/static",
		"/v1/config/public",
		"/v1/featureflags",
		"/v1/metadata",
		"/v1/mypermissions",
		"/v1/telemetry/config",
	}
)

// Handler handles HTTP proxy requests to Central.
type Handler struct {
	proxy            *httputil.ReverseProxy
	centralReachable atomic.Bool
	authorizer       *k8sAuthorizer
}

// NewProxyHandler creates a new proxy handler that forwards requests to Central.
func NewProxyHandler(centralEndpoint string, centralCertificates []*x509.Certificate, token string) (*Handler, error) {
	centralBaseURL, err := url.Parse(
		urlfmt.FormatURL(centralEndpoint, urlfmt.HTTPS, urlfmt.NoTrailingSlash),
	)
	if err != nil {
		return nil, errors.Wrap(err, "parsing endpoint")
	}

	proxy, err := newCentralReverseProxy(centralBaseURL, centralCertificates, token)
	if err != nil {
		return nil, errors.Wrap(err, "creating central reverse proxy")
	}

	restConfig, err := k8sutil.GetK8sInClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "getting in-cluster config")
	}
	retryablehttp.ConfigureRESTConfig(restConfig)

	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "creating kubernetes client")
	}

	return &Handler{
		proxy:      proxy,
		authorizer: newK8sAuthorizer(k8sClient),
	}, nil
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

	return nil
}

// isAuthzSkipPath checks if the request path matches any of the authorization-skip patterns.
// These paths still require authentication but skip the authorization (SubjectAccessReview) check.
// Matching enforces segment boundaries: path must equal pattern exactly or start with pattern + "/".
// This prevents "/v1/metadata" from matching "/v1/metadataExtra".
func isAuthzSkipPath(requestPath string) bool {
	// Normalize the path before authorization checks to prevent bypass via path manipulation
	// (e.g., double slashes, dot segments).
	normalizedPath := path.Clean(requestPath)

	for _, pattern := range authzSkipPathPrefixes {
		if normalizedPath == pattern || strings.HasPrefix(normalizedPath, pattern+"/") {
			return true
		}
	}
	return false
}

// ServeHTTP handles incoming HTTP requests and proxies them to Central.
func (h *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if err := h.validateRequest(request); err != nil {
		pkghttputil.WriteError(writer, err)
		return
	}

	if h.authorizer == nil {
		log.Error("Authorizer is nil - this indicates a misconfiguration in the central proxy handler")
		pkghttputil.WriteError(writer, pkghttputil.NewError(http.StatusInternalServerError, "authorizer not configured"))
		return
	}

	// Require authentication for all endpoints.
	userInfo, err := h.authorizer.authenticate(request.Context(), request)
	if err != nil {
		pkghttputil.WriteError(writer, err)
		return
	}

	if !isAuthzSkipPath(request.URL.Path) {
		if err := h.authorizer.authorize(request.Context(), userInfo, request); err != nil {
			pkghttputil.WriteError(writer, err)
			return
		}
	}

	h.proxy.ServeHTTP(writer, request)
}
