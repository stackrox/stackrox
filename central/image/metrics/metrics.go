package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	ImageScanSemaphoreQueueSize = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "image_scan_semaphore_queue_size",
		Help:      "A counter that tracks the queue size of the scan semaphre used in image scan.",
	})
)

func init() {
	prometheus.MustRegister(ImageScanSemaphoreQueueSize)
}
