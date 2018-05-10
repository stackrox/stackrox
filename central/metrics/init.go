package metrics

import (
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/prometheus/client_golang/prometheus"
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
