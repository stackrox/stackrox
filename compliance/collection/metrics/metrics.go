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
		Buckets:   []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 15, 20, 50, 100, 500, 1000},
	},
		[]string{
			// The Node this scan belongs to
			"node_name",
			// Whether the inventory run was completed successfully
			"error",
		})

	callToNodeInventoryDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "call_node_inventory_duration_seconds",
		Help:      "Time between sending the request to Node Inventory and getting the reply in Compliance (per Node) in seconds",
		Buckets:   []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 15, 20, 50, 100, 500, 1000},
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
		Name:      "protobuf_inventory_message_size_bytes",
		Help:      "Message size of sent Node Inventory gRPC messages (per Node) in bytes",
		Buckets:   []float64{500, 1000, 10000, 50000, 100000, 500000, 1000000},
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

	inventoryTransmissions = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "inventory_transmissions_total",
		Help:      "Number of node inventory scans sent to sensor",
	},
		[]string{
			// The Node this scan belongs to
			"node_name",
			"transmission_type",
		})
)

// ObserveNodeInventoryScan observes the metric.
func ObserveNodeInventoryScan(inventory *storage.NodeInventory) {
	rhelPackageCount := 0
	components := inventory.GetComponents()

	if components == nil {
		return
	}

	if components.GetRhelComponents() != nil {
		rhelPackageCount = len(components.GetRhelComponents())
	}
	numberOfRHELPackages.With(prometheus.Labels{
		"node_name":    inventory.GetNodeName(),
		"os_namespace": components.GetNamespace(),
	}).Set(float64(rhelPackageCount))

	rhelContentSets := 0
	if components.GetRhelContentSets() != nil {
		rhelContentSets = len(components.GetRhelContentSets())
	}
	numberOfContentSets.With(prometheus.Labels{
		"node_name":    inventory.GetNodeName(),
		"os_namespace": components.GetNamespace(),
	}).Set(float64(rhelContentSets))
}

// ObserveNodeInventoryCallDuration observes the metric.
func ObserveNodeInventoryCallDuration(d time.Duration, nodeName string, e error) {
	callToNodeInventoryDuration.With(prometheus.Labels{
		"node_name": nodeName,
		"error":     strconv.FormatBool(e != nil),
	}).Observe(d.Seconds())
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
		"node_name": cmsg.GetNode(),
	}).Observe(float64(cmsg.Size()))
}

type InventoryTransmission string

const (
	InventoryTransmissionScan               InventoryTransmission = "scanning"
	InventoryTransmissionResendingCacheHit  InventoryTransmission = "resending cached"
	InventoryTransmissionResendingCacheMiss InventoryTransmission = "scanning and resending "
)

// ObserveNodeInventorySending observes the metric.
func ObserveNodeInventorySending(nodeName string, sendingType InventoryTransmission) {
	inventoryTransmissions.With(prometheus.Labels{
		"node_name":         nodeName,
		"transmission_type": string(sendingType),
	}).Inc()
}

func init() {
	prometheus.MustRegister(numberOfRHELPackages, numberOfContentSets, scanDuration, callToNodeInventoryDuration, rescanInterval, scansTotal, protobufMessageSize)
}
