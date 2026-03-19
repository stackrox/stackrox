package centralproxy

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

// Result label values for proxyRequestsTotal and proxyRequestDuration.
const (
	requestResultSuccess         = "success"
	requestResultNotImplemented  = "not_implemented"
	requestResultValidationError = "validation_error"
	requestResultConfigError     = "config_error"
	requestResultAuthnError      = "authn_error"
	requestResultAuthzError      = "authz_error"
	requestResultProxyError      = "proxy_error"
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
	// proxyRequestsTotal counts proxy requests by result.
	proxyRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "central_proxy_requests_total",
		Help:      "Total number of proxy requests to Central by result.",
	}, []string{"result"})

	// proxyRequestDuration tracks proxy request latency by result.
	proxyRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "central_proxy_request_duration_seconds",
		Help:      "Duration of proxy requests to Central in seconds by result.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"result"})

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
	metrics.EmplaceCollector(
		proxyRequestsTotal,
		proxyRequestDuration,
		proxyAuthenticationTotal,
		proxyAuthorizationTotal,
		proxyTokenRequestsTotal,
	)
}

// observeProxyRequest increments the request counter and observes the request duration.
func observeProxyRequest(result string, duration time.Duration) {
	proxyRequestsTotal.WithLabelValues(result).Inc()
	proxyRequestDuration.WithLabelValues(result).Observe(duration.Seconds())
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
