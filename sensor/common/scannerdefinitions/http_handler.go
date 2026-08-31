package scannerdefinitions

import (
	"crypto/x509"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralclient"
	"google.golang.org/grpc/codes"
)

const scannerDefsPath = "/api/extensions/scannerdefinitions"

const (
	// repo2CPEFileParam matches sensorMappingsFile in compliance/node/index/indexer.go.
	repo2CPEFileParam = "repo2cpe"

	// repo2CPERefreshInterval is the steady-state cadence other repo2cpe consumers use.
	repo2CPERefreshInterval = 4 * time.Hour
	// repo2CPERetryInterval is short so a cold cache reaches its first success quickly.
	repo2CPERetryInterval = time.Minute
	repo2CPEFetchTimeout  = 30 * time.Second

	etagHeader            = "ETag"
	ifNoneMatchHeader     = "If-None-Match"
	lastModifiedHeader    = "Last-Modified"
	ifModifiedSinceHeader = "If-Modified-Since"
)

var (
	headersToProxy = set.NewFrozenStringSet("If-Modified-Since", "Accept-Encoding")
	log            = logging.LoggerForModule()
)

// Handler handles requests to retrieve scanner definitions
// from Central.
type Handler struct {
	centralClient    *http.Client
	centralReachable atomic.Bool
}

// NewDefinitionsHandler creates a new scanner definitions handler and a
// repo-to-CPE refresher that share Central's HTTP client.
func NewDefinitionsHandler(centralEndpoint string, centralCertificates []*x509.Certificate) (*Handler, *Repo2CPE, error) {
	client, err := centralclient.AuthenticatedCentralHTTPClient(centralEndpoint, centralCertificates)
	if err != nil {
		return nil, nil, errors.Wrap(err, "instantiating central HTTP transport")
	}
	return &Handler{centralClient: client}, NewRepo2CPE(client), nil
}

// Notify reacts to sensor going into online/offline mode.
func (h *Handler) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e, "Scanner definitions handler"))
	switch e {
	case common.SensorComponentEventCentralReachable:
		h.centralReachable.Store(true)
	case common.SensorComponentEventOfflineMode:
		h.centralReachable.Store(false)
	}
}

func (h *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// Validate request.
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// If central is not reachable, then the request should return an error to Scanner.
	if !h.centralReachable.Load() {
		httputil.WriteGRPCStyleErrorf(writer, codes.Unavailable, "central not reachable")
		return
	}

	// Prepare the Central's request, proxy relevant headers and all parameters.
	// No need to set Scheme nor Host, as the client will already do that for us.
	centralURL := url.URL{
		Path:     scannerDefsPath,
		RawQuery: request.URL.RawQuery,
	}
	centralRequest, err := http.NewRequestWithContext(
		request.Context(), http.MethodGet, centralURL.String(), nil)
	if err != nil {
		httputil.WriteGRPCStyleErrorf(writer, codes.Internal, "failed to create request: %v", err)
		return
	}
	// Proxy relevant headers.
	for headerName := range headersToProxy.All() {
		for _, value := range request.Header.Values(headerName) {
			centralRequest.Header.Add(headerName, value)
		}
	}
	// Do request, copy all response headers, and body.
	resp, err := h.centralClient.Do(centralRequest)
	if err != nil {
		httputil.WriteGRPCStyleErrorf(writer, codes.Internal, "failed to contact central: %v", err)
		return
	}
	defer utils.IgnoreError(resp.Body.Close)
	for k, vs := range resp.Header {
		for _, v := range vs {
			writer.Header().Add(k, v)
		}
	}
	writer.WriteHeader(resp.StatusCode)
	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		httputil.WriteGRPCStyleErrorf(writer, codes.Internal, "failed write response: %v", err)
		return
	}
}
