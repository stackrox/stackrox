package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	// general
	prometheus.MustRegister(panicCounter)
	prometheus.MustRegister(sensorIndicatorChannelFullCounter)
	prometheus.MustRegister(totalNetworkFlowsSentCounter)
	prometheus.MustRegister(processDedupeCacheMisses)
}
