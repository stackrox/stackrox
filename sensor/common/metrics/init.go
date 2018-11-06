package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	// general
	prometheus.MustRegister(
		processDedupeCacheHits,
		processDedupeCacheMisses,
		panicCounter,
		sensorIndicatorChannelFullCounter,
		totalNetworkFlowsSentCounter,
		totalNetworkFlowsReceivedCounter,
		sensorEvents,
	)
}
