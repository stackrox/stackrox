package scannerdefinitions

import (
	"io"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/clientconn"
	"github.com/stackrox/stackrox/pkg/httputil"
	"github.com/stackrox/stackrox/pkg/mtls"
	"github.com/stackrox/stackrox/pkg/set"
	"github.com/stackrox/stackrox/pkg/utils"
	"google.golang.org/grpc/codes"
)

var (
	headersToProxy = set.NewFrozenStringSet("If-Modified-Since", "Accept-Encoding")
)

// scannerDefinitionsHandler handles requests to retrieve scanner definitions
// from Central.
type scannerDefinitionsHandler struct {
	centralClient *http.Client
	centralHost   string
}

// NewDefinitionsHandler creates a new scanner definitions handler.
func NewDefinitionsHandler(centralHost string) (http.Handler, error) {
	client, err := clientconn.NewHTTPClient(mtls.CentralSubject, centralHost, 0)
	if err != nil {
		return nil, errors.Wrap(err, "instantiating central HTTP transport")
	}
	return &scannerDefinitionsHandler{
		centralClient: client,
		centralHost:   centralHost,
	}, nil
}

func (h *scannerDefinitionsHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// Validate request.
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	// Prepare the Central's request, proxy relevant headers and all parameters.
	centralURL := url.URL{
		Scheme:   "https",
		Host:     h.centralHost,
		Path:     "api/extensions/scannerdefinitions",
		RawQuery: request.URL.Query().Encode(),
	}
	centralRequest, err := http.NewRequestWithContext(
		request.Context(), http.MethodGet, centralURL.String(), nil)
	if err != nil {
		httputil.WriteGRPCStyleErrorf(writer, codes.Internal, "failed to create request: %v", err)
		return
	}
	// Proxy relevant headers.
	for _, headerName := range headersToProxy.AsSlice() {
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
