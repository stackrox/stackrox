package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	numberOfRHELPackages = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "num_packages_in_inventory",
		Help:      "Number of packages discovered by the last Node Inventory (per Node)",
	},
		[]string{
			// The Node this scan belongs to
			"node_name",
		})

	scanTime = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "inventory_scan_time",
		Help:      "Scan time for Node Inventory (per Node) in seconds",
	},
		[]string{
			// The Node this scan belongs to
			"node_name",
		})
)

// ObserveNodeInventoryScan observes the metric.
func ObserveNodeInventoryScan(inventory *storage.NodeInventory, scanDuration time.Duration) {
	rhelPackageCount := 0
	if inventory.Components.RhelComponents != nil {
		rhelPackageCount = len(inventory.Components.RhelComponents)
	}

	numberOfRHELPackages.With(prometheus.Labels{
		"node_name": inventory.NodeName,
	}).Set(float64(rhelPackageCount))

	scanTime.With(prometheus.Labels{
		"node_name": inventory.NodeName,
	}).Observe(scanDuration.Seconds())
}

func init() {
	prometheus.MustRegister(numberOfRHELPackages, scanTime)
}
