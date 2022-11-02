package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	listFeaturesDurationMilliseconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "list_features_duration_millis",
		Help:    "Time it takes to list all the features in a layer.",
		Buckets: []float64{1, 10, 100, 500, 1000},
	}, []string{"packageformat", "step"})
)

func init() {
	prometheus.MustRegister(listFeaturesDurationMilliseconds)
}

// ObserveListFeaturesTime observes `ListFeatures` for the given package format and sub-step from the given start time.
func ObserveListFeaturesTime(packageformat, step string, start time.Time) {
	listFeaturesDurationMilliseconds.
		WithLabelValues(packageformat, step).
		Observe(float64(time.Since(start).Nanoseconds()) / float64(time.Millisecond))
}
