package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
)

var (
	alpnEndpointMetric = prometheus.NewCounterVec(
		// Leaving subsystem empty to skip duplicating this metric for each component
		prometheus.CounterOpts{
			Namespace: pkgMetrics.PrometheusNamespace,
			Subsystem: "", // empty, so fqName = rox_endpoint_tls_handshakes_with_negotiated_alp_total
			Name:      "endpoint_tls_handshakes_with_negotiated_alp_total",
			Help:      "Number of finished TLS handshakes ...",
		},
		[]string{"subsystem", "endpoint", "remoteAddr", "alp"},
	)
)

func init() {
	prometheus.MustRegister(alpnEndpointMetric)
}

func ObserveALPN(sub, endpointAddr, remoteAddr, alp string) {
	alpnEndpointMetric.WithLabelValues(sub, endpointAddr, remoteAddr, alp).Inc()
}
