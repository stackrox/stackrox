package centralproxy

import (
	"crypto/x509"
	"net/http"
	"net/http/httputil"
	"net/url"
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

// ServeHTTP handles incoming HTTP requests and proxies them to Central.
func (h *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if err := h.validateRequest(request); err != nil {
		pkghttputil.WriteError(writer, err)
		return
	}

	if h.authorizer != nil {
		if err := h.authorizer.authorize(request.Context(), request); err != nil {
			pkghttputil.WriteError(writer, err)
			return
		}
	}

	h.proxy.ServeHTTP(writer, request)
}
