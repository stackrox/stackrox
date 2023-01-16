package metrics

import (
	"fmt"
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
			"nodeName",
			// The number of installed RHEL packages discovered by the scan
			"numRHELPackages",
		})

	scanTime = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "inventory_scan_time_seconds",
		Help:      "Scan time for Node Inventory (per Node)",
	},
		[]string{
			// The time it took to complete the Node Inventory scan on a given node
			"scanTime",
		})
)

// ObserveNodeInventoryScan observes the metric.
func ObserveNodeInventoryScan(inventory *storage.NodeInventory) {
	rhelPackageCount := 0
	if inventory.Components.RhelComponents != nil {
		rhelPackageCount = len(inventory.Components.RhelComponents)
	}

	numberOfRHELPackages.With(prometheus.Labels{
		"nodeName":        inventory.NodeName,
		"numRHELPackages": fmt.Sprintf("%d", rhelPackageCount),
	})
}

// ObserveNodeInventoryScanTime observes the metric.
func ObserveNodeInventoryScanTime(duration time.Duration) {
	scanTime.With(prometheus.Labels{
		"scanTime": fmt.Sprintf("%d", duration),
	})
}

func init() {
	prometheus.MustRegister(numberOfRHELPackages, scanTime)
}
