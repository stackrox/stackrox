package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	// general
	prometheus.MustRegister(processDedupeCacheHits)
	prometheus.MustRegister(processDedupeCacheMisses)
	prometheus.MustRegister(panicCounter)
	prometheus.MustRegister(sensorIndicatorChannelFullCounter)
	prometheus.MustRegister(totalNetworkFlowsSentCounter)
	prometheus.MustRegister(totalNetworkFlowsReceivedCounter)
}
