package manager

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	cacheResultHit     = "hit"
	cacheResultMiss    = "miss"
	cacheResultExpired = "expired"
	cacheResultSkip    = "skip"

	reviewResultAllowed  = "allowed"
	reviewResultDenied   = "denied"
	reviewResultBypassed = "bypassed"
	reviewResultError    = "error"

	fetchSourceSensor  = "sensor"
	fetchSourceCentral = "central"

	fetchStatusSuccess = "success"
	fetchStatusTimeout = "timeout"
	fetchStatusError   = "error"
)

var (
	imageCacheOperations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.AdmissionControlSubsystem.String(),
		Name:      "image_cache_operations_total",
		Help:      "Total image cache lookups.",
	}, []string{"result"})

	imageFetchTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.AdmissionControlSubsystem.String(),
		Name:      "image_fetch_total",
		Help:      "Total image fetch RPCs issued to Sensor or Central.",
	}, []string{"source"})

	imageFetchDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.AdmissionControlSubsystem.String(),
		Name:      "image_fetch_duration_seconds",
		Help:      "Duration of individual image fetch RPCs issued to Sensor or Central.",
		Buckets:   prometheus.ExponentialBuckets(0.05, 2, 9), // 50ms to ~12.8s
	}, []string{"source", "status"})

	policyevalReviewDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.AdmissionControlSubsystem.String(),
		Name:      "policyeval_review_duration_seconds",
		Help:      "End-to-end duration of deploy time policy enforcement admission review.",
		Buckets:   prometheus.ExponentialBuckets(0.005, 2, 12), // 5ms to ~10s
	}, []string{"result"})

	policyevalReviewTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.AdmissionControlSubsystem.String(),
		Name:      "policyeval_review_total",
		Help:      "Total deploy time policy enforcement admission reviews processed.",
	}, []string{"result"})

	imageFetchesPerReview = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.AdmissionControlSubsystem.String(),
		Name:      "image_fetches_per_review",
		Help:      "Number of image fetch RPCs issued per admission review.",
		Buckets:   prometheus.LinearBuckets(0, 1, 11), // 0 to 10
	})
)

func init() {
	prometheus.MustRegister(
		imageCacheOperations,
		imageFetchTotal,
		imageFetchDuration,
		imageFetchesPerReview,
		policyevalReviewDuration,
		policyevalReviewTotal,
	)
}

func observeCacheHit() {
	imageCacheOperations.WithLabelValues(cacheResultHit).Inc()
}

func observeCacheMiss() {
	imageCacheOperations.WithLabelValues(cacheResultMiss).Inc()
}

func observeCacheExpired() {
	imageCacheOperations.WithLabelValues(cacheResultExpired).Inc()
}

// observeCacheSkip records lookups bypassed because the image has no ID to use as cache key.
func observeCacheSkip() {
	imageCacheOperations.WithLabelValues(cacheResultSkip).Inc()
}

func observeImageFetch(source string, duration time.Duration, err error) {
	fetchStatus := fetchStatusSuccess
	if err != nil {
		if status.Code(err) == codes.DeadlineExceeded {
			fetchStatus = fetchStatusTimeout
		} else {
			fetchStatus = fetchStatusError
		}
	}
	imageFetchTotal.WithLabelValues(source).Inc()
	imageFetchDuration.WithLabelValues(source, fetchStatus).Observe(duration.Seconds())
}

func observeAdmissionReview(result string, duration time.Duration) {
	policyevalReviewTotal.WithLabelValues(result).Inc()
	if result != reviewResultBypassed {
		policyevalReviewDuration.WithLabelValues(result).Observe(duration.Seconds())
	}
}

func observeImageFetchesPerReview(count int) {
	imageFetchesPerReview.Observe(float64(count))
}
