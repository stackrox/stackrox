package centralproxy

import (
	"crypto/x509"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"

	"github.com/pkg/errors"
	pkghttputil "github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/sensor/common"
	"google.golang.org/grpc/codes"
)

var log = logging.LoggerForModule()

// Handler handles HTTP proxy requests to Central.
type Handler struct {
	proxy            *httputil.ReverseProxy
	centralReachable atomic.Bool
}

// NewProxyHandler creates a new proxy handler that forwards requests to Central.
func NewProxyHandler(centralEndpoint string, centralCertificates []*x509.Certificate, token string) (*Handler, error) {
	centralBaseURL, err := url.Parse(
		urlfmt.FormatURL(centralEndpoint, urlfmt.HTTPS, urlfmt.NoTrailingSlash),
	)
	if err != nil {
		return nil, errors.Wrap(err, "parsing endpoint")
	}

	transport, err := newHTTPTransportWithToken(centralBaseURL, centralCertificates, token)
	if err != nil {
		return nil, errors.Wrap(err, "creating HTTP transport with token")
	}

	return &Handler{
		proxy: newProxy(centralBaseURL, transport),
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

// validateRequest validates the incoming request and writes appropriate error responses.
func (h *Handler) validateRequest(writer http.ResponseWriter, request *http.Request) bool {
	if request.Method != http.MethodGet && request.Method != http.MethodPost {
		pkghttputil.WriteGRPCStyleErrorf(writer, codes.Unimplemented, "method %s not allowed", request.Method)
		return false
	}

	if !h.centralReachable.Load() {
		pkghttputil.WriteGRPCStyleErrorf(writer, codes.Unavailable, "central not reachable")
		return false
	}

	return true
}

// ServeHTTP handles incoming HTTP requests and proxies them to Central.
func (h *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if !h.validateRequest(writer, request) {
		return
	}
	h.proxy.ServeHTTP(writer, request)
}
