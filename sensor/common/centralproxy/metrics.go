package centralproxy

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	proxyRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "central_proxy_request_duration_seconds",
		Help:      "Duration of requests proxied to Central",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method", "status_code"})

	tokenRequestDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "central_proxy_token_request_duration_seconds",
		Help:      "Duration of internal token requests to Central",
		Buckets:   prometheus.DefBuckets,
	})
)

func init() {
	prometheus.MustRegister(proxyRequestDuration, tokenRequestDuration)
}
