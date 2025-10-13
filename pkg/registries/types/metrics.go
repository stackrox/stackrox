package types

import (
	"context"
	"errors"
	"math"
	"net/http"
	"os"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stackrox/rox/pkg/metrics"
)

// MetricsHandler wraps the registry Prometheus metrics.
type MetricsHandler struct {
	requestCounter    *prometheus.CounterVec
	timeoutCounter    *prometheus.CounterVec
	durationHistogram *prometheus.HistogramVec
}

// roundedDurationBuckets produces exponential buckets rounded to the next millisecond.
// The smallest bucket contains all durations smaller than 0.1 seconds. The largest finite
// bucket is set to 20 seconds. The total number of buckets is 20.
// Note that depending on env.RegistryClientTimeout, the request duration may fall
// outside of the bucket scope. In this case, the request will fall into the "infinite"
// bucket.
func roundedDurationBuckets() []float64 {
	buckets := prometheus.ExponentialBucketsRange(0.1, 20, 20)
	for i := range buckets {
		buckets[i] = math.Round(buckets[i]*1000) / 1000
	}
	return buckets
}

// NewMetricsHandler creates a new metrics handler.
func NewMetricsHandler(subsystem metrics.Subsystem) *MetricsHandler {
	requestCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metrics.PrometheusNamespace,
			Subsystem: subsystem.String(),
			Name:      "registry_client_requests_total",
			Help:      "The number of registry requests per count and method.",
		},
		[]string{"code", "method", "type"},
	)
	timeoutCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metrics.PrometheusNamespace,
			Subsystem: subsystem.String(),
			Name:      "registry_client_error_timeouts_total",
			Help:      "The number of registry timeout errors.",
		},
		[]string{"type"},
	)
	durationHistogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metrics.PrometheusNamespace,
			Subsystem: subsystem.String(),
			Name:      "registry_client_request_duration_seconds",
			Help:      "A histogram of registry response times.",
			Buckets:   roundedDurationBuckets(),
		},
		[]string{"type"},
	)
	prometheus.MustRegister(requestCounter, timeoutCounter, durationHistogram)
	return &MetricsHandler{
		requestCounter:    requestCounter,
		timeoutCounter:    timeoutCounter,
		durationHistogram: durationHistogram,
	}
}

func instrumentRoundTripperTimeout(counter *prometheus.CounterVec, next http.RoundTripper,
	registryType string,
) promhttp.RoundTripperFunc {
	return func(r *http.Request) (*http.Response, error) {
		resp, err := next.RoundTrip(r)
		if errors.Is(err, os.ErrDeadlineExceeded) {
			counter.With(prometheus.Labels{"type": registryType}).Inc()
		}
		return resp, err
	}
}

// RoundTripper returns a transport that is instrumented with Prometheus metrics.
func (m *MetricsHandler) RoundTripper(base http.RoundTripper, registryType string) http.RoundTripper {
	if m == nil {
		return base
	}
	labelOpt := promhttp.WithLabelFromCtx("type",
		func(_ context.Context) string {
			return registryType
		},
	)
	return promhttp.InstrumentRoundTripperCounter(m.requestCounter,
		instrumentRoundTripperTimeout(m.timeoutCounter,
			promhttp.InstrumentRoundTripperDuration(m.durationHistogram, base, labelOpt),
			registryType,
		),
		labelOpt,
	)
}

func (m *MetricsHandler) TestCollectRequestCounter(t *testing.T) int {
	return testutil.CollectAndCount(m.requestCounter)
}

func (m *MetricsHandler) TestCollectTimeoutCounter(t *testing.T) int {
	return testutil.CollectAndCount(m.timeoutCounter)
}

func (m *MetricsHandler) TestCollectHistogramCounter(t *testing.T) int {
	return testutil.CollectAndCount(m.durationHistogram)
}
