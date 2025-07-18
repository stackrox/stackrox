package tlscheckcache

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	tlsCheckCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: "", // empty as this is shared among multiple subsystems.
		Name:      "tls_check_count",
		Help:      "The total number of TLS checks requested",
	}, []string{"subsystem"})

	tlsCheckDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: "", // empty as this is shared among multiple subsystems.
		Name:      "tls_check_duration_seconds",
		Help:      "Time taken in seconds to perform a TLS check",
	}, []string{"subsystem"})
)

// incrementTLSCheckCount adds to the total count of TLS check requests made via the registry store.
func incrementTLSCheckCount(subsystem metrics.Subsystem) {
	tlsCheckCount.WithLabelValues(subsystem.String()).Inc()
}

// observeTLSCheckDuration observes the time in seconds taken to perform a TLS check.
func observeTLSCheckDuration(subsystem metrics.Subsystem, t time.Duration) {
	tlsCheckDuration.WithLabelValues(subsystem.String()).Observe(t.Seconds())
}

func init() {
	prometheus.MustRegister(
		tlsCheckCount,
		tlsCheckDuration,
	)
}
