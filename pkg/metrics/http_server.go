package metrics

import (
	"net/http"
	"time"

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
	Address      string
	Gatherer     prometheus.Gatherer
	HandlerOpts  promhttp.HandlerOpts
	uptimeMetric prometheus.Gauge
}

// NewDefaultHTTPServer creates and returns a new metrics http server with configured settings.
func NewDefaultHTTPServer(subsystem Subsystem) *HTTPServer {
	if err := env.ValidateMetricsSetting(); err != nil {
		utils.Should(errors.Wrap(err, "invalid metrics port setting"))
		log.Error(errors.Wrap(err, "metrics server is disabled"))
		return nil
	}
	if !env.MetricsEnabled() {
		log.Warn("Metrics server is disabled")
		return nil
	}

	uptimeMetric := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: PrometheusNamespace,
		Subsystem: subsystem.String(),
		Name:      "uptime_seconds",
		Help:      "Total number of seconds that the service has been up",
	})
	// Allow the metric to be registered multiple times for tests
	_ = prometheus.Register(uptimeMetric)

	return &HTTPServer{
		Address:      env.MetricsSetting.Setting(),
		Gatherer:     prometheus.DefaultGatherer,
		uptimeMetric: uptimeMetric,
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
	go gatherUptimeMetricForever(time.Now(), s.uptimeMetric)
}

func gatherUptimeMetricForever(startTime time.Time, uptimeMetric prometheus.Gauge) {
	t := time.NewTicker(5 * time.Second)
	for range t.C {
		uptimeMetric.Set(time.Since(startTime).Seconds())
	}
}

func runForever(server *http.Server) {
	err := server.ListenAndServe()
	// The HTTP server should never terminate.
	log.Panicf("Unexpected termination of metrics HTTP server: %v", err)
}
