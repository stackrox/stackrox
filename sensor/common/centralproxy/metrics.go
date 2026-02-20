package centralproxy

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

// Outcome label values for proxyRequestsTotal and proxyRequestDuration.
const (
	outcomeSuccess         = "success"
	outcomeNotImplemented  = "not_implemented"
	outcomeValidationError = "validation_error"
	outcomeConfigError     = "config_error"
	outcomeAuthnError      = "authn_error"
	outcomeAuthzError      = "authz_error"
)

// Result label values for proxyAuthenticationTotal.
const (
	authnResultSuccess  = "success"
	authnResultCacheHit = "cache_hit"
	authnResultError    = "error"
)

// Result label values for proxyAuthorizationTotal.
const (
	authzResultSuccess  = "success"
	authzResultCacheHit = "cache_hit"
	authzResultDenied   = "denied"
	authzResultSkipped  = "skipped"
	authzResultError    = "error"
)

// Result label values for proxyTokenRequestsTotal.
const (
	tokenResultSuccess  = "success"
	tokenResultCacheHit = "cache_hit"
	tokenResultError    = "error"
)

var (
	// proxyRequestsTotal counts proxy requests by outcome.
	proxyRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "central_proxy_requests_total",
		Help:      "Total number of proxy requests to Central by outcome.",
	}, []string{"outcome"})

	// proxyRequestDuration tracks proxy request latency by outcome.
	proxyRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "central_proxy_request_duration_seconds",
		Help:      "Duration of proxy requests to Central in seconds by outcome.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"outcome"})

	// proxyAuthenticationTotal counts authentication attempts by result.
	proxyAuthenticationTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "central_proxy_authentication_total",
		Help:      "Total number of authentication attempts by result.",
	}, []string{"result"})

	// proxyAuthorizationTotal counts authorization attempts by result.
	proxyAuthorizationTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "central_proxy_authorization_total",
		Help:      "Total number of authorization attempts by result.",
	}, []string{"result"})

	// proxyTokenRequestsTotal counts token acquisition requests by result.
	proxyTokenRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "central_proxy_token_requests_total",
		Help:      "Total number of token requests to Central by result.",
	}, []string{"result"})
)

func init() {
	prometheus.MustRegister(
		proxyRequestsTotal,
		proxyRequestDuration,
		proxyAuthenticationTotal,
		proxyAuthorizationTotal,
		proxyTokenRequestsTotal,
	)
}

// observeProxyRequest increments the request counter and observes the request duration.
func observeProxyRequest(outcome string, duration time.Duration) {
	proxyRequestsTotal.WithLabelValues(outcome).Inc()
	proxyRequestDuration.WithLabelValues(outcome).Observe(duration.Seconds())
}

// incrementAuthentication increments the authentication counter for the given result.
func incrementAuthentication(result string) {
	proxyAuthenticationTotal.WithLabelValues(result).Inc()
}

// incrementAuthorization increments the authorization counter for the given result.
func incrementAuthorization(result string) {
	proxyAuthorizationTotal.WithLabelValues(result).Inc()
}

// incrementTokenRequest increments the token request counter for the given result.
func incrementTokenRequest(result string) {
	proxyTokenRequestsTotal.WithLabelValues(result).Inc()
}
