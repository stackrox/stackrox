package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	EnricherSemaphoreQueueSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "enricher_semaphore_queue_size",
		Help:      "A counter that tracks the size of the queues for the scan semaphores used in image scans.",
	})
	EnricherSemaphoreHoldingSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "enricher_semaphore_holding_size",
		Help:      "A counter that tracks the number of requests successfully holding scanner semaphores.",
	})
)

func init() {
	prometheus.MustRegister(EnricherSemaphoreQueueSize,
		EnricherSemaphoreHoldingSize)
}
