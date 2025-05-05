package images

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	ScanSemaphoreQueueSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "image_scan_semaphore_queue_size",
		Help:      "A gauge vector that tracks the size of the queue for the scan semaphores used in scans.",
	}, []string{"location"})
	ScanSemaphoreHoldingSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "image_scan_semaphore_holding_size",
		Help:      "A gauge vector that tracks the number of requests successfully holding the scan semaphores.",
	}, []string{"location"})
	ScanSemaphoreLimit = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "image_scan_semaphore_limit",
		Help:      "A gauge vector that tracks the limit of available scan semaphores.",
	}, []string{"location"})
)

func init() {
	prometheus.MustRegister(ScanSemaphoreQueueSize,
		ScanSemaphoreHoldingSize,
		ScanSemaphoreLimit)
}
