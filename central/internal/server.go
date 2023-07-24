package internal

import (
	"log"
	"net/http"

	"github.com/NYTimes/gziphandler"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/metrics"
	"github.com/stackrox/rox/pkg/mtls"
)

// HTTPServer is a HTTP server to serve functionality available only within the cluster.
type HTTPServer struct {
	Address string
	mux     *http.ServeMux
	metrics metrics.HTTPMetrics
}

// NewHTTPServer creates and returns a new cluster-internal http server.
func NewHTTPServer(metricsInstance metrics.HTTPMetrics) *HTTPServer {
	return &HTTPServer{
		Address: env.ClusterInternalPortSetting.Setting(),
		mux:     http.NewServeMux(),
		metrics: metricsInstance,
	}
}

// AddRoutes adds routes to cluster-internal server to be exposed on.
func (s *HTTPServer) AddRoutes(routes []*Route) {
	for _, r := range routes {
		h := r.ServerHandler
		if r.Compression {
			h = gziphandler.GzipHandler(h)
		}
		if s.metrics != nil {
			h = s.metrics.WrapHandler(h, r.Route)
		}
		s.mux.Handle(r.Route, h)
	}
}

// RunForever starts the HTTP server in the background.
func (s *HTTPServer) RunForever() {
	httpServer := &http.Server{
		Addr:    s.Address,
		Handler: s.mux,
	}
	go runForever(httpServer)
}

func runForever(server *http.Server) {
	// The reason we reuse mTLS certificate here is
	// we only use TLS certificate here for encryption and thus
	// do not care about which identity this certificate presents.
	// Alternative solution discussed included issuing separate certificate for this endpoint
	// which we decided not to pursue due to much bigger complexity.
	err := server.ListenAndServeTLS(mtls.CertFilePath(), mtls.KeyFilePath())
	// The HTTP server should never terminate.
	log.Panicf("Unexpected termination of private HTTP server: %v", err)
}
