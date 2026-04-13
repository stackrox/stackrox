package app

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/sensor/admission-control/manager"
)

func initMetrics() {
	prometheus.MustRegister(
		manager.ImageCacheOperations,
		manager.ImageFetchTotal,
		manager.ImageFetchDuration,
		manager.ImageFetchesPerReview,
		manager.PolicyevalReviewDuration,
		manager.PolicyevalReviewTotal,
	)
}
