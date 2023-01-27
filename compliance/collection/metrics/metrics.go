package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	numberOfRHELPackages = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "packages_in_inventory",
		Help:      "Number of packages discovered by the last Node Inventory (per Node)",
	},
		[]string{
			// The Node this scan belongs to
			"node_name",
			// The OS name and version of the Node
			"os_namespace",
		})

	numberOfContentSets = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "content_sets_in_inventory",
		Help:      "Number of content sets discovered by the last Node Inventory (per Node)",
	},
		[]string{
			// The Node this scan belongs to
			"node_name",
			// The OS name and version of the Node
			"os_namespace",
		})

	scanDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "inventory_scan_duration_seconds",
		Help:      "Scan duration for Node Inventory (per Node) in seconds",
	},
		[]string{
			// The Node this scan belongs to
			"node_name",
			// Whether the inventory run was completed successfully
			"error",
		})

	rescanInterval = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "rescan_interval_seconds",
		Help:      "Time in seconds between Node Inventory runs",
	},
		[]string{
			// The Node this scan belongs to
			"node_name",
		})

	protobufMessageSize = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "protobuf_inventory_message_size",
		Help:      "Message size of sent Node Inventory gRPC messages (per Node) in bytes",
	},
		[]string{
			// The Node this scan belongs to
			"node_name",
		})

	scansTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "inventory_scans_total",
		Help:      "Number of run node inventory scans since container start",
	},
		[]string{
			// The Node this scan belongs to
			"node_name",
		})
)

// ObserveNodeInventoryScan observes the metric.
func ObserveNodeInventoryScan(inventory *storage.NodeInventory) {
	rhelPackageCount := 0
	if inventory.Components.RhelComponents != nil {
		rhelPackageCount = len(inventory.Components.RhelComponents)
	}
	numberOfRHELPackages.With(prometheus.Labels{
		"node_name":    inventory.NodeName,
		"os_namespace": inventory.Components.Namespace,
	}).Set(float64(rhelPackageCount))

	rhelContentSets := 0
	if inventory.Components.RhelContentSets != nil {
		rhelContentSets = len(inventory.Components.RhelContentSets)
	}
	numberOfContentSets.With(prometheus.Labels{
		"node_name":    inventory.NodeName,
		"os_namespace": inventory.Components.Namespace,
	}).Set(float64(rhelContentSets))
}

// ObserveScanDuration observes the metric.
func ObserveScanDuration(d time.Duration, nodeName string, e error) {
	scanDuration.With(prometheus.Labels{
		"node_name": nodeName,
		"error":     strconv.FormatBool(e != nil),
	}).Observe(d.Seconds())
}

// ObserveRescanInterval observes the metric.
func ObserveRescanInterval(d time.Duration, nodeName string) {
	rescanInterval.With(prometheus.Labels{
		"node_name": nodeName,
	}).Set(d.Seconds())
}

// ObserveScansTotal observed the metric
func ObserveScansTotal(nodeName string) {
	scansTotal.With(prometheus.Labels{
		"node_name": nodeName,
	}).Inc()
}

// ObserveInventoryProtobufMessage observes the metric.
func ObserveInventoryProtobufMessage(cmsg *sensor.MsgFromCompliance) {
	protobufMessageSize.With(prometheus.Labels{
		"node_name": cmsg.Node,
	}).Observe(float64(cmsg.Size()))
}

// TODO(ROX-13164): Add number of retries

func init() {
	prometheus.MustRegister(numberOfRHELPackages, numberOfContentSets, scanDuration, rescanInterval, scansTotal, protobufMessageSize)
}
