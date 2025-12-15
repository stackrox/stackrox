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
		PerSensorRate,
		PerSensorBucketCapacity,
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

	// RequestsAccepted tracks requests accepted by the rate limiter (per sensor).
	RequestsAccepted = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metrics.PrometheusNamespace,
			Subsystem: metrics.CentralSubsystem.String(),
			Name:      "rate_limiter_requests_accepted_total",
			Help:      "Requests accepted by the rate limiter",
		},
		[]string{"workload", "sensor_id"},
	)

	// RequestsRejected tracks requests rejected by the rate limiter (per sensor).
	RequestsRejected = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metrics.PrometheusNamespace,
			Subsystem: metrics.CentralSubsystem.String(),
			Name:      "rate_limiter_requests_rejected_total",
			Help:      "Requests rejected by the rate limiter",
		},
		[]string{"workload", "sensor_id", "reason"},
	)

	// PerSensorRate tracks the current per-sensor rate limit (requests per second).
	PerSensorRate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metrics.PrometheusNamespace,
			Subsystem: metrics.CentralSubsystem.String(),
			Name:      "rate_limiter_per_sensor_bucket_refill_rate_per_second",
			Help: "Current per-sensor rate limit in requests per second. " +
				"This is also the rate at which tokens are refilled.",
		},
		[]string{"workload"},
	)

	// PerSensorBucketCapacity tracks the current per-sensor token bucket capacity (max tokens).
	PerSensorBucketCapacity = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metrics.PrometheusNamespace,
			Subsystem: metrics.CentralSubsystem.String(),
			Name:      "rate_limiter_per_sensor_bucket_max_tokens",
			Help: "Current per-sensor token bucket capacity (max tokens). Must be a positive integer. " +
				"This is the maximum number of requests that can be accepted in a burst before rate limiting kicks in.",
		},
		[]string{"workload"},
	)
)
