package centralproxy

import (
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/centralsensor"
	pkghttputil "github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/retryablehttp"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
)

var (
	log = logging.LoggerForModule()

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
	retryablehttp.ConfigureRESTConfig(restConfig)

	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "creating kubernetes client")
	}

	return &Handler{
		clusterIDGetter: clusterIDGetter,
		authorizer:      newK8sAuthorizer(k8sClient),
		transport:       transport,
		proxy:           proxy,
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
	if err := checkInternalTokenAPISupport(); err != nil {
		http.Error(writer, err.Error(), pkghttputil.StatusFromError(err))
		return
	}

	if err := h.validateRequest(request); err != nil {
		http.Error(writer, err.Error(), pkghttputil.StatusFromError(err))
		return
	}

	if h.authorizer == nil {
		log.Error("Authorizer is nil - this indicates a misconfiguration in the central proxy handler")
		http.Error(writer, "authorizer not configured", http.StatusInternalServerError)
		return
	}

	userInfo, err := h.authorizer.authenticate(request.Context(), request)
	if err != nil {
		http.Error(writer, err.Error(), pkghttputil.StatusFromError(err))
		return
	}

	if err := h.authorizer.authorize(request.Context(), userInfo, request); err != nil {
		http.Error(writer, err.Error(), pkghttputil.StatusFromError(err))
		return
	}

	h.proxy.ServeHTTP(writer, request)
}
