package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
)

var (
	alpnEndpointMetrics = make(map[string]*prometheus.CounterVec)
)

func init() {
	// Only Central and Sensor expose an endpoint that demultiplexes connections based on ALPN.
	for _, subsystem := range []pkgMetrics.Subsystem{pkgMetrics.CentralSubsystem, pkgMetrics.SensorSubsystem} {
		metric := newALPMetricForSub(subsystem.String())
		alpnEndpointMetrics[subsystem.String()] = metric
		prometheus.MustRegister(metric)
	}
}

func newALPMetricForSub(sub string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: pkgMetrics.PrometheusNamespace,
		Subsystem: sub,
		Name:      "endpoint_tls_handshakes_with_negotiated_alp_total",
		Help: "Number of finished TLS handshakes to a given endpoint with " +
			"given Application Level Protocol being negotiated as a result of ALPN.",
	}, []string{"endpoint", "remoteAddr", "alp"})
}

func ObserveALPN(sub, endpointAddr, remoteAddr, alp string) {
	if metric, found := alpnEndpointMetrics[sub]; found {
		metric.WithLabelValues(endpointAddr, remoteAddr, alp).Inc()
	}
}
