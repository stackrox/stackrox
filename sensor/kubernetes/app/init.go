package app

import (
	"github.com/stackrox/rox/sensor/common/metrics"
	listenerresourcesmetrics "github.com/stackrox/rox/sensor/kubernetes/listener/resources/metrics"
)

func initMetrics() {
	metrics.Init()
	listenerresourcesmetrics.Init()
}
