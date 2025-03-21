package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	SensorScanSemaphoreQueueSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "sensor_scan_semaphore_queue_size",
		Help:      "A counter that tracks the size of the queue for the scan semaphore used in sensor scan.",
	})
	SensorScanSemaphoreHoldingSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "sensor_scan_semaphore_holding_size",
		Help:      "A counter that tracks the number of requests successfully holding the semaphore.",
	})
)

func init() {
	prometheus.MustRegister(SensorScanSemaphoreQueueSize,
		SensorScanSemaphoreHoldingSize)
}
