package metrics

import (
	"net/http"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	metricsURLPath = "/metrics"
)

var (
	log = logging.LoggerForModule()
)

// HTTPServer is a HTTP server for exporting Prometheus metrics.
type HTTPServer struct {
	Address     string
	Gatherer    prometheus.Gatherer
	HandlerOpts promhttp.HandlerOpts
}

// NewDefaultHTTPServer creates and returns a new metrics http server with configured settings.
func NewDefaultHTTPServer() *HTTPServer {
	if err := env.ValidateMetricsSetting(); err != nil {
		utils.Should(errors.Wrap(err, "invalid metrics port setting"))
		log.Error(errors.Wrap(err, "metrics server is disabled"))
		return nil
	}
	if !env.MetricsEnabled() {
		log.Warn("Metrics server is disabled")
		return nil
	}

	return &HTTPServer{
		Address:  env.MetricsSetting.Setting(),
		Gatherer: prometheus.DefaultGatherer,
	}
}

// RunForever starts the HTTP server in the background.
func (s *HTTPServer) RunForever() {
	if s == nil {
		return
	}
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
