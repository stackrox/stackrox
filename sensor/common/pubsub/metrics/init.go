package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	prometheus.MustRegister(
		laneEventOperations,
		laneQueueSize,
		laneEventProcessingDuration,
		consumersCount,
	)
}
