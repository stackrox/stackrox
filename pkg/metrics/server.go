package metrics

import (
	"context"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
	"go.uber.org/zap"
)

const (
	metricsURLPath = "/metrics"
)

var log = logging.LoggerForModule()

// Server is a HTTP server for exporting Prometheus metrics.
type Server struct {
	metricsServer       *http.Server
	secureMetricsServer *http.Server
	tlsConfigLoader     *tlsConfigLoader
	uptimeMetric        prometheus.Gauge
}

// NewServer creates and returns a new metrics http(s) server with configured settings.
func NewServer(subsystem Subsystem) *Server {
	mux := http.NewServeMux()
	mux.Handle(metricsURLPath, promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{}))

	var metricsServer *http.Server
	if metricsEnabled() {
		metricsServer = &http.Server{
			Addr:    env.MetricsPort.Setting(),
			Handler: mux,
		}
	}

	var tlsConfigLoader *tlsConfigLoader
	var secureMetricsServer *http.Server
	if secureMetricsEnabled() {
		tlsConfigLoader = createTLSConfigLoader()
		tlsConfig, err := tlsConfigLoader.TLSConfig()
		if err != nil {
			utils.Should(errors.Wrap(err, "failed to create TLS config loader"))
			return nil
		}
		secureMetricsServer = &http.Server{
			Addr:      env.SecureMetricsPort.Setting(),
			Handler:   mux,
			TLSConfig: tlsConfig,
		}
	}

	uptimeMetric := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: PrometheusNamespace,
		Subsystem: subsystem.String(),
		Name:      "uptime_seconds",
		Help:      "Total number of seconds that the service has been up",
	})
	// Allow the metric to be registered multiple times for tests
	_ = prometheus.Register(uptimeMetric)

	return &Server{
		metricsServer:       metricsServer,
		secureMetricsServer: secureMetricsServer,
		tlsConfigLoader:     tlsConfigLoader,
		uptimeMetric:        uptimeMetric,
	}
}

// RunForever starts the HTTP and HTTPS server in the background.
func (s *Server) RunForever() {
	if s == nil {
		return
	}

	runMetrics := metricsEnabled() && metricsValid()
	if runMetrics {
		go runForever(s.metricsServer)
	}

	runSecureMetrics := secureMetricsEnabled() && s.secureMetricsValid()
	if runSecureMetrics {
		s.tlsConfigLoader.WatchForChanges()
		go runForeverTLS(s.secureMetricsServer)
	}

	if runMetrics || runSecureMetrics {
		go gatherUptimeMetricForever(time.Now(), s.uptimeMetric)
	}
}

func (s *Server) Stop(ctx context.Context) {
	if s == nil {
		return
	}

	if metricsEnabled() {
		if err := s.metricsServer.Shutdown(ctx); err != nil {
			log.Errorw("Failed to shutdown metrics server", zap.Error(err))
			s.metricsServer.Close()
		}
	}
	if secureMetricsEnabled() {
		if err := s.secureMetricsServer.Shutdown(ctx); err != nil {
			log.Errorw("Failed to shutdown secure metrics server", zap.Error(err))
			s.secureMetricsServer.Close()
		}
	}
}

func metricsEnabled() bool {
	if !env.MetricsEnabled() {
		log.Warn("Metrics server is disabled")
		return false
	}
	return true
}

func metricsValid() bool {
	if err := env.ValidateMetricsSetting(); err != nil {
		utils.Should(errors.Wrap(err, "invalid metrics setting"))
		log.Error(errors.Wrap(err, "metrics server is disabled"))
		return false
	}
	return true
}

func secureMetricsEnabled() bool {
	if !env.SecureMetricsEnabled() {
		log.Warn("Secure metrics server is disabled")
		return false
	}
	return true
}

func (s *Server) secureMetricsValid() bool {
	if err := env.ValidateSecureMetricsSetting(); err != nil {
		utils.Should(errors.Wrap(err, "invalid secure metrics setting"))
		log.Error(errors.Wrap(err, "secure metrics server is disabled"))
		return false
	}
	if s.tlsConfigLoader == nil {
		utils.Should(errors.New("invalid TLS config loader"))
		return false
	}
	return true
}

func createTLSConfigLoader() *tlsConfigLoader {
	if !secureMetricsEnabled() {
		return nil
	}

	certDir := env.SecureMetricsCertDir.Setting()
	clientCANamespace := env.SecureMetricsClientCANamespace.Setting()
	clientCAConfigMap := env.SecureMetricsClientCAConfigMap.Setting()
	tlsConfigLoader, err := NewTLSConfigLoader(certDir, clientCANamespace, clientCAConfigMap)
	if err != nil {
		log.Error(errors.Wrap(err, "failed to create TLS config loader"))
	}
	return tlsConfigLoader
}

func gatherUptimeMetricForever(startTime time.Time, uptimeMetric prometheus.Gauge) {
	t := time.NewTicker(5 * time.Second)
	for range t.C {
		uptimeMetric.Set(time.Since(startTime).Seconds())
	}
}

func runForever(server *http.Server) {
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		// The HTTP server should never terminate.
		log.Panicf("Unexpected termination of metrics server %q: %v", server.Addr, err)
	}
}

func runForeverTLS(server *http.Server) {
	if err := server.ListenAndServeTLS(certFilePath(), keyFilePath()); !errors.Is(err, http.ErrServerClosed) {
		// The HTTPS server should never terminate.
		log.Panicf("Unexpected termination of secure metrics server %q: %v", server.Addr, err)
	}
}
