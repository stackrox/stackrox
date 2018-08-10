package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	logger = logging.LoggerForModule()
)

func init() {
	// general
	prometheus.MustRegister(panicCounter)
	prometheus.MustRegister(boltOperationHistogramVec)
	prometheus.MustRegister(indexOperationHistogramVec)
}
