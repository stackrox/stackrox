package images

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	ScanSemaphoreHoldingSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: "", // empty as this is shared among multiple subsystems.
		Name:      "image_scan_semaphore_holding_size",
		Help:      "A gauge vector that shows the number of scan tasks currently holding the scan semaphores.",
	}, []string{"subsystem", "entity", "requestedFrom"})
	ScanSemaphoreQueueSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: "", // empty to match the other metrics in this file that share the labels.
		Name:      "image_scan_semaphore_queue_size",
		Help:      "A gauge that shows how many scan tasks are currently waiting for a semaphore to be available.",
	}, []string{"subsystem", "entity", "requestedFrom"})
	scanSemaphoreLimit = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: "", // empty to match the other metrics in this file that share the labels.
		Name:      "image_scan_semaphore_limit",
		Help: "A gauge that shows the maximum number of available scan semaphores. " +
			"It is configured through env and can only change on pod restart.",
	}, []string{"subsystem", "entity", "requestedFrom"})
)

func SetSensorScanSemaphoreLimit(limit float64, forRequestsFrom string) {
	scanSemaphoreLimit.WithLabelValues("sensor", "delegated-scan", forRequestsFrom).Set(limit)
}

func SetCentralScanSemaphoreLimit(limit float64) {
	scanSemaphoreLimit.WithLabelValues("central", "central-image-scan-service", "n/a").Set(limit)
}

func init() {
	prometheus.MustRegister(ScanSemaphoreQueueSize,
		ScanSemaphoreHoldingSize,
		scanSemaphoreLimit)
}
