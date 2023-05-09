package metrics

import (
	"context"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/quay/zlog"
)

const (
	defaultAddress = ":9091"
	metricsURLPath = "/v4/metrics"
)

// HTTPServer is an HTTP server for exporting Prometheus metrics.
type HTTPServer struct {
	server *http.Server
}

// NewHTTPServer creates a new metrics HTTP server with the configured settings.
//
// This function uses a default port of 9090 if the given port is empty.
// It returns nil if the port <= 0.
func NewHTTPServer(port string) *HTTPServer {
	addr := defaultAddress
	if port != "" {
		if port == "0" || port[0] == '-' {
			return nil
		}
		addr = ":" + port
	}

	mux := http.NewServeMux()
	mux.Handle(metricsURLPath, promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{}))

	return &HTTPServer{
		server: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
	}
}

// Start starts the HTTP server in the background.
func (s *HTTPServer) Start(ctx context.Context) {
	if s == nil {
		return
	}

	go gatherThrottleMetrics(ctx)

	err := s.server.ListenAndServe()
	// The metrics HTTP server should never terminate.
	zlog.Error(ctx).
		Err(err).
		Msg("Unexpected termination of metrics server")
}
