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
		PerClientRate,
		PerClientBucketCapacity,
		PerClientBucketTokens,
	)
}

var (
	// RequestsTotal tracks all requests received by the rate limiter with outcome.
	// Use this for overall volume visibility. Per-client detail is available in logs.
	RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metrics.PrometheusNamespace,
			Subsystem: metrics.CentralSubsystem.String(),
			Name:      "rate_limiter_requests_total",
			Help:      "Total requests received by the rate limiter",
		},
		[]string{"workload", "outcome"},
	)

	// ActiveClients tracks the current number of active clients for each workload.
	ActiveClients = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metrics.PrometheusNamespace,
			Subsystem: metrics.CentralSubsystem.String(),
			Name:      "rate_limiter_active_clients",
			Help:      "Current number of active clients being rate limited",
		},
		[]string{"workload"},
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

	// PerClientBucketTokens tracks the current number of tokens available in each client's bucket.
	PerClientBucketTokens = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metrics.PrometheusNamespace,
			Subsystem: metrics.CentralSubsystem.String(),
			Name:      "rate_limiter_per_client_bucket_tokens",
			Help: "Current number of tokens available in each client's bucket. " +
				"Tokens are consumed on each request and refilled at the configured rate. " +
				"When tokens reach zero, requests are rejected until tokens are refilled.",
		},
		[]string{"workload", "cluster_id"},
	)
)
