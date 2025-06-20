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
		Help:      "A gauge that shows how many scan tasks are currently waiting for a semaphore to be available.",
	}, []string{"entity"})
	ScanSemaphoreHoldingSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "image_scan_semaphore_holding_size",
		Help:      "A gauge vector that shows the number of scan tasks currently holding the scan semaphores.",
	}, []string{"entity"})
	ScanSemaphoreLimit = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "image_scan_semaphore_limit",
		Help: "A gauge that shows the maximum number of available scan semaphores. " +
			"It is configured through env and can only change on pod restart.",
	}, []string{"entity"})
)

func init() {
	prometheus.MustRegister(ScanSemaphoreQueueSize,
		ScanSemaphoreHoldingSize,
		ScanSemaphoreLimit)
}
