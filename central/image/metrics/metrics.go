package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	ImageScanSemaphoreQueueSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "image_scan_semaphore_queue_size",
		Help:      "A counter that tracks the size of the queue for the scan semaphore used in image scan.",
	})
	ImageScanSemaphoreHoldingSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "image_scan_semaphore_holding_size",
		Help:      "A counter that tracks the number of requests successfully holding the semaphore.",
	})
	ImageScanSemaphoreLimit = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "image_scan_semaphore_limit",
		Help:      "A counter that tracks the limit of available scan semaphores.",
	})
)

func init() {
	prometheus.MustRegister(ImageScanSemaphoreQueueSize,
		ImageScanSemaphoreHoldingSize,
		ImageScanSemaphoreLimit)
}
