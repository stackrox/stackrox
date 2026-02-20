package service

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

// Result label values for tokenGenerationTotal.
const (
	tokenGenResultSuccess           = "success"
	tokenGenResultInvalidArgs       = "invalid_args"
	tokenGenResultRoleCreationError = "role_creation_error"
	tokenGenResultIssuanceError     = "token_issuance_error" // #nosec G101 not a hardcoded credential
)

var (
	// tokenGenerationTotal counts token generation attempts by result.
	tokenGenerationTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "internal_token_generation_total",
		Help:      "Total number of internal token generation attempts by result.",
	}, []string{"result"})

	// tokenGenerationDuration tracks the latency of token generation attempts by result.
	tokenGenerationDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "internal_token_generation_duration_seconds",
		Help:      "Duration of internal token generation attempts in seconds by result.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"result"})
)

func init() {
	prometheus.MustRegister(
		tokenGenerationTotal,
		tokenGenerationDuration,
	)
}

// observeTokenGeneration increments the token generation counter and observes
// the generation duration for all results.
func observeTokenGeneration(result string, duration time.Duration) {
	tokenGenerationTotal.WithLabelValues(result).Inc()
	tokenGenerationDuration.WithLabelValues(result).Observe(duration.Seconds())
}
