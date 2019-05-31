package metrics

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	defaultAddress = "localhost:9090"
	metricsURLPath = "/metrics"
)

// HTTPServer is a HTTP server for exporting Prometheus metrics.
type HTTPServer struct {
	Address     string
	Gatherer    prometheus.Gatherer
	HandlerOpts promhttp.HandlerOpts
}

// NewDefaultHTTPServer creates and returns a new metrics http server with default settings.
func NewDefaultHTTPServer() *HTTPServer {
	return &HTTPServer{
		Address:  defaultAddress,
		Gatherer: prometheus.DefaultGatherer,
	}
}

// RunForever starts the HTTP server in the background.
func (s *HTTPServer) RunForever() {
	mux := http.NewServeMux()
	mux.Handle(metricsURLPath, promhttp.HandlerFor(s.Gatherer, s.HandlerOpts))
	httpServer := &http.Server{
		Addr:    s.Address,
		Handler: mux,
	}
	go runForever(httpServer)
}

func runForever(server *http.Server) {
	err := server.ListenAndServe()
	// The HTTP server should never terminate.
	log.Panicf("Unexpected termination of metrics HTTP server: %v", err)
}
