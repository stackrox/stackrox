package enricher

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
)

// This interface encapsulates the metrics this package needs.
type metrics interface {
	SetScanDurationTime(start time.Time, scanner string, err error)
	SetNodeInventoryNumberComponents(count int, clusterName string, nodeName string)
}

type metricsImpl struct {
	scanTimeDuration           *prometheus.HistogramVec
	nodeInventoryComponentSize *prometheus.GaugeVec
}

func startTimeToMS(t time.Time) float64 {
	return float64(time.Since(t).Nanoseconds()) / float64(time.Millisecond)
}

func (m *metricsImpl) SetScanDurationTime(start time.Time, scanner string, err error) {
	m.scanTimeDuration.With(prometheus.Labels{"Scanner": scanner, "Error": fmt.Sprintf("%t", err != nil)}).Observe(startTimeToMS(start))
}

func (m *metricsImpl) SetNodeInventoryNumberComponents(count int, clusterName string, nodeName string) {
	m.nodeInventoryComponentSize.WithLabelValues(clusterName, nodeName).Set(float64(count))
}

func newMetrics(subsystem pkgMetrics.Subsystem) metrics {
	m := &metricsImpl{
		scanTimeDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: pkgMetrics.PrometheusNamespace,
			Subsystem: subsystem.String(),
			Name:      "node_scan_duration",
			Help:      "Amount of time it's taken to scan a node in ms",
			Buckets:   prometheus.ExponentialBuckets(4, 2, 16),
		}, []string{"Scanner", "Error"}),
		nodeInventoryComponentSize: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: pkgMetrics.PrometheusNamespace,
			Subsystem: subsystem.String(),
			Name:      "node_scan_num_components",
			Help:      "Number of discovered components per Node",
		}, []string{"ClusterName", "NodeName"}),
	}

	pkgMetrics.EmplaceCollector(
		m.scanTimeDuration,
		m.nodeInventoryComponentSize,
	)

	return m
}
