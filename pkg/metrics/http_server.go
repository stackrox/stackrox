package metrics

import (
	"crypto/tls"
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

var log = logging.LoggerForModule()

// MetricsServer is a HTTP server for exporting Prometheus metrics.
type MetricsServer struct {
	Address         string
	SecureAddress   string
	Gatherer        prometheus.Gatherer
	HandlerOpts     promhttp.HandlerOpts
	tlsConfigLoader *tlsConfigLoader
	uptimeMetric    prometheus.Gauge
}

// NewMetricsServer creates and returns a new metrics http(s) server with configured settings.
func NewMetricsServer(subsystem Subsystem) *MetricsServer {
	uptimeMetric := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: PrometheusNamespace,
		Subsystem: subsystem.String(),
		Name:      "uptime_seconds",
		Help:      "Total number of seconds that the service has been up",
	})
	// Allow the metric to be registered multiple times for tests
	_ = prometheus.Register(uptimeMetric)

	certDir := env.SecureMetricsCertDir.Setting()
	clientCANamespace := env.SecureMetricsClientCANamespace.Setting()
	clientCAConfigMap := env.SecureMetricsClientCAConfigMap.Setting()
	// TODO: handle error here
	tlsLoader, _ := NewTLSConfigLoader(certDir, clientCANamespace, clientCAConfigMap)
	return &MetricsServer{
		Address:         env.MetricsPort.Setting(),
		SecureAddress:   env.SecureMetricsPort.Setting(),
		Gatherer:        prometheus.DefaultGatherer,
		tlsConfigLoader: tlsLoader,
		uptimeMetric:    uptimeMetric,
	}
}

// RunForever starts the HTTP and HTTPS server in the background.
func (s *MetricsServer) RunForever() {
	if s == nil {
		return
	}
	mux := http.NewServeMux()
	mux.Handle(metricsURLPath, promhttp.HandlerFor(s.Gatherer, s.HandlerOpts))

	metricsEnabled := metricsEnabled()
	if metricsEnabled {
		go runForever(s.Address, mux)
	}

	secureMetricsEnabled := secureMetricsEnabled()
	if secureMetricsEnabled {
		s.tlsConfigLoader.WatchForChanges()
		tlsConfig := s.tlsConfigLoader.TLSConfig
		go runForeverTLS(s.SecureAddress, mux, tlsConfig)
	}

	if metricsEnabled || secureMetricsEnabled {
		go gatherUptimeMetricForever(time.Now(), s.uptimeMetric)
	}
}

func metricsEnabled() bool {
	if err := env.ValidateMetricsSetting(); err != nil {
		utils.Should(errors.Wrap(err, "invalid metrics setting"))
		log.Error(errors.Wrap(err, "metrics server is disabled"))
		return false
	}
	if !env.MetricsEnabled() {
		log.Warn("Metrics server is disabled")
		return false
	}
	return true
}

func secureMetricsEnabled() bool {
	if err := env.ValidateSecureMetricsSetting(); err != nil {
		utils.Should(errors.Wrap(err, "invalid secure metrics setting"))
		log.Error(errors.Wrap(err, "secure metrics server is disabled"))
		return false
	}
	if !env.SecureMetricsEnabled() {
		log.Warn("Secure metrics server is disabled")
		return false
	}
	return true
}

func gatherUptimeMetricForever(startTime time.Time, uptimeMetric prometheus.Gauge) {
	t := time.NewTicker(5 * time.Second)
	for range t.C {
		uptimeMetric.Set(time.Since(startTime).Seconds())
	}
}

func runForever(address string, mux *http.ServeMux) {
	server := &http.Server{
		Addr:    address,
		Handler: mux,
	}
	err := server.ListenAndServe()
	// The HTTP server should never terminate.
	log.Panicf("Unexpected termination of metrics server %q: %v", server.Addr, err)
}

func runForeverTLS(address string, mux *http.ServeMux, tlsConfig *tls.Config) {
	server := &http.Server{
		Addr:      address,
		Handler:   mux,
		TLSConfig: tlsConfig,
	}
	err := server.ListenAndServeTLS("", "")
	// The HTTPS server should never terminate.
	log.Panicf("Unexpected termination of metrics server %q: %v", server.Addr, err)
}
