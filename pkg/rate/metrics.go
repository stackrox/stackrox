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
		RequestsAccepted,
		RequestsRejected,
		PerClientRate,
		PerClientBucketCapacity,
	)
}

var (
	// RequestsTotal tracks all requests received by the rate limiter with outcome.
	RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metrics.PrometheusNamespace,
			Subsystem: metrics.CentralSubsystem.String(),
			Name:      "rate_limiter_requests_total",
			Help:      "Total requests received by the rate limiter",
		},
		[]string{"workload", "outcome"},
	)

	// RequestsAccepted tracks requests accepted by the rate limiter (per client).
	RequestsAccepted = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metrics.PrometheusNamespace,
			Subsystem: metrics.CentralSubsystem.String(),
			Name:      "rate_limiter_requests_accepted_total",
			Help:      "Requests accepted by the rate limiter",
		},
		[]string{"workload", "client_id"},
	)

	// RequestsRejected tracks requests rejected by the rate limiter (per client).
	RequestsRejected = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metrics.PrometheusNamespace,
			Subsystem: metrics.CentralSubsystem.String(),
			Name:      "rate_limiter_requests_rejected_total",
			Help:      "Requests rejected by the rate limiter",
		},
		[]string{"workload", "client_id", "reason"},
	)

	// PerClientRate tracks the current per-client rate limit (requests per second).
	PerClientRate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metrics.PrometheusNamespace,
			Subsystem: metrics.CentralSubsystem.String(),
			Name:      "rate_limiter_per_client_bucket_refill_rate_per_second",
			Help: "Current per-client rate limit in requests per second. " +
				"This is also the rate at which tokens are refilled.",
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
				"This is the maximum number of requests that can be accepted in a burst before rate limiting kicks in.",
		},
		[]string{"workload"},
	)
)
