package rate

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

const (
	// OutcomeAccepted is the outcome label value for accepted requests.
	OutcomeAccepted = "accepted"
	// OutcomeRejected is the outcome label value for rejected requests.
	OutcomeRejected = "rejected"
)

func init() {
	prometheus.MustRegister(
		RequestsTotal,
		ActiveClients,
		InFlightTokens,
		PerClientBucketCapacity,
	)
}

var (
	// RequestsTotal tracks all requests received by the limiter with outcome.
	RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metrics.PrometheusNamespace,
			Subsystem: metrics.CentralSubsystem.String(),
			Name:      "rate_limiter_requests_total",
			Help:      "Total requests received by the limiter",
		},
		[]string{"workload", "outcome"},
	)

	// ActiveClients tracks the current number of active clients for each workload.
	ActiveClients = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metrics.PrometheusNamespace,
			Subsystem: metrics.CentralSubsystem.String(),
			Name:      "rate_limiter_active_clients",
			Help:      "Current number of active clients being limited",
		},
		[]string{"workload"},
	)

	// InFlightTokens tracks the number of tokens currently consumed and not yet returned.
	InFlightTokens = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metrics.PrometheusNamespace,
			Subsystem: metrics.CentralSubsystem.String(),
			Name:      "rate_limiter_in_flight_tokens",
			Help: "Number of tokens currently consumed (in-flight). " +
				"A token is consumed when a request is accepted and returned when processing completes.",
		},
		[]string{"workload"},
	)

	// PerClientBucketCapacity tracks the current per-client token bucket capacity (max tokens).
	PerClientBucketCapacity = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metrics.PrometheusNamespace,
			Subsystem: metrics.CentralSubsystem.String(),
			Name:      "rate_limiter_per_client_bucket_max_tokens",
			Help: "Current per-client token bucket capacity (max tokens). Must be a positive integer. " +
				"This is the maximum number of requests that can be in-flight concurrently per client.",
		},
		[]string{"workload"},
	)
)
