package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/metrics"
)

const (
	ScannerVersionV2 = "Stackrox Scanner"
	ScannerVersionV4 = "Scanner V4"
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
			// The version of Scanner this metric was generated for
			"scanner_version",
		})

	numberOfReportPackages = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "packages_in_package_report",
		Help:      "Number of packages discovered by the last v2 Node Inventory or v4 IndexReport (per Node)",
	},
		[]string{
			// The Node this scan belongs to
			"node_name",
			// The OS name and version of the Node
			"os_namespace",
			// The version of Scanner this metric was generated for
			"scanner_version",
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
			// The version of Scanner this metric was generated for
			"scanner_version",
		})

	numberOfReportContentSets = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "content_sets_in_package_report",
		Help:      "Number of content sets discovered by the last v2 Node Inventory or v4 Node Index (per Node)",
	},
		[]string{
			// The Node this scan belongs to
			"node_name",
			// The OS name and version of the Node
			"os_namespace",
			// The version of Scanner this metric was generated for
			"scanner_version",
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
			// The version of Scanner this metric was generated for
			"scanner_version",
		})

	indexDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "index_duration_seconds",
		Help:      "Generation duration for Node IndexReports (per Node) in seconds",
		Buckets:   []float64{0.1, 0.5, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 15, 20, 50, 100, 500, 1000},
	},
		[]string{
			// The Node this scan belongs to
			"node_name",
			// Whether the inventory run was completed successfully
			"error",
			// The version of Scanner this metric was generated for
			"scanner_version",
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
			// The version of Scanner this metric was generated for
			"scanner_version",
		})

	rescanInterval = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "rescan_interval_seconds",
		Help:      "Time in seconds between runs (identical for v2 Node Inventories and v4 Index Reports)",
	},
		[]string{
			// The Node this scan belongs to
			"node_name",
			// The version of Scanner this metric was generated for
			"scanner_version",
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
			// The version of Scanner this metric was generated for
			"scanner_version",
		})

	protobufReportMessageSize = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "protobuf_report_message_size_bytes",
		Help:      "Message size of sent v2 Node Inventory or v4 Node Index gRPC messages (per Node) in bytes",
		Buckets:   []float64{500, 1000, 10000, 50000, 100000, 500000, 1000000},
	},
		[]string{
			// The Node this scan belongs to
			"node_name",
			// The version of Scanner this metric was generated for
			"scanner_version",
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
			// The version of Scanner this metric was generated for
			"scanner_version",
		})

	indexesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "index_reports_total",
		Help:      "Number of generated node index reports since container start",
	},
		[]string{
			// The Node this scan belongs to
			"node_name",
			// The version of Scanner this metric was generated for
			"scanner_version",
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
			// The version of Scanner this metric was generated for
			"scanner_version",
		})

	// nodePackageReportTransmissions is the new version of inventoryTransmissions that carries a more generic name.
	// inventoryTransmissions is kept for backwards compatibility so we still get metrics from older clusters.
	nodePackageReportTransmissions = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "node_package_reports_total",
		Help:      "Number of total v2 node inventory scans and v4 index reports sent to sensor",
	},
		[]string{
			// The Node this scan belongs to
			"node_name",
			"transmission_type",
			// The version of Scanner this metric was generated for
			"scanner_version",
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
	observePackages(inventory.GetNodeName(), components.GetNamespace(), ScannerVersionV2, rhelPackageCount)

	rhelContentSets := 0
	if components.GetRhelContentSets() != nil {
		rhelContentSets = len(components.GetRhelContentSets())
	}
	observeContentSets(inventory.GetNodeName(), components.GetNamespace(), ScannerVersionV2, rhelContentSets)
}

// ObserveNodeIndexReport observes the metric for Scanner v4.
func ObserveNodeIndexReport(report *v4.IndexReport, nodeName string) {
	rhelPackageCount := 0
	contents := report.GetContents()

	if contents == nil {
		return
	}

	if contents.GetPackages() != nil {
		rhelPackageCount = len(contents.GetPackages())
	}
	observePackages(nodeName, "", ScannerVersionV4, rhelPackageCount)

	rhelContentSets := 0
	if contents.GetRepositories() != nil {
		rhelContentSets = len(contents.GetRepositories())
	}
	observeContentSets(nodeName, "", ScannerVersionV4, rhelContentSets)
}

func observePackages(nodeName string, osNamespace string, scannerVersion string, packageCount int) {
	numberOfReportPackages.With(prometheus.Labels{
		"node_name":       nodeName,
		"os_namespace":    osNamespace,
		"scanner_version": scannerVersion,
	}).Set(float64(packageCount))
	// Record the old metric for backwards compatibility
	// Remove the block below after Scanner v2 is out of support for clusters
	numberOfRHELPackages.With(prometheus.Labels{
		"node_name":       nodeName,
		"os_namespace":    osNamespace,
		"scanner_version": scannerVersion,
	}).Set(float64(packageCount))
}

func observeContentSets(nodeName string, osNamespace string, scannerVersion string, contentSetCount int) {
	numberOfReportContentSets.With(prometheus.Labels{
		"node_name":       nodeName,
		"os_namespace":    osNamespace,
		"scanner_version": scannerVersion,
	}).Set(float64(contentSetCount))
	// Record the old metric for backwards compatibility
	// Remove the block below after Scanner v2 is out of support for clusters
	numberOfContentSets.With(prometheus.Labels{
		"node_name":       nodeName,
		"os_namespace":    osNamespace,
		"scanner_version": scannerVersion,
	}).Set(float64(contentSetCount))
}

// ObserveNodeInventoryCallDuration observes the metric.
func ObserveNodeInventoryCallDuration(d time.Duration, nodeName string, e error) {
	callToNodeInventoryDuration.With(prometheus.Labels{
		"node_name":       nodeName,
		"error":           strconv.FormatBool(e != nil),
		"scanner_version": ScannerVersionV2,
	}).Observe(d.Seconds())
}

// ObserveScanDuration observes the metric.
func ObserveScanDuration(d time.Duration, nodeName string, e error) {
	scanDuration.With(prometheus.Labels{
		"node_name":       nodeName,
		"error":           strconv.FormatBool(e != nil),
		"scanner_version": ScannerVersionV2,
	}).Observe(d.Seconds())
}

// ObserveIndexDuration observes the metric.
func ObserveIndexDuration(d time.Duration, nodeName string, e error) {
	indexDuration.With(prometheus.Labels{
		"node_name":       nodeName,
		"error":           strconv.FormatBool(e != nil),
		"scanner_version": ScannerVersionV4,
	}).Observe(d.Seconds())
}

// ObserveRescanInterval observes the metric.
func ObserveRescanInterval(d time.Duration, nodeName, scannerVersion string) {
	rescanInterval.With(prometheus.Labels{
		"node_name":       nodeName,
		"scanner_version": scannerVersion,
	}).Set(d.Seconds())
}

// ObserveScansTotal observed the metric
func ObserveScansTotal(nodeName string) {
	scansTotal.With(prometheus.Labels{
		"node_name":       nodeName,
		"scanner_version": ScannerVersionV2,
	}).Inc()
}

// ObserveIndexesTotal observed the metric
func ObserveIndexesTotal(nodeName string) {
	indexesTotal.With(prometheus.Labels{
		"node_name":       nodeName,
		"scanner_version": ScannerVersionV4,
	}).Inc()
}

// ObserveReportProtobufMessage observes the metric.
func ObserveReportProtobufMessage(cmsg *sensor.MsgFromCompliance, scannerVersion string) {
	protobufReportMessageSize.With(prometheus.Labels{
		"node_name":       cmsg.GetNode(),
		"scanner_version": scannerVersion,
	}).Observe(float64(cmsg.SizeVT()))
	// Record the old metric for backwards compatibility
	// Remove the block below after Scanner v2 is out of support for clusters
	protobufMessageSize.With(prometheus.Labels{
		"node_name":       cmsg.GetNode(),
		"scanner_version": scannerVersion,
	}).Observe(float64(cmsg.SizeVT()))
}

// InventoryTransmission names the way in which a NodeInventory was obtained
type InventoryTransmission string

const (
	// InventoryTransmissionScan means that we requested a new scan from NodeInventory container
	InventoryTransmissionScan InventoryTransmission = "scanning"
	// InventoryTransmissionResendingCacheHit means that we reply to NACK and send NodeInventory from compliance cache
	InventoryTransmissionResendingCacheHit InventoryTransmission = "resending cached"
	// InventoryTransmissionResendingCacheMiss means that we reply to NACK and schedule a rescan due to empty cache.
	// This will result in additional observation of `InventoryTransmissionScan`
	InventoryTransmissionResendingCacheMiss InventoryTransmission = "scanning and resending "
)

// ObserveNodePackageReportTransmissions observes the metric.
func ObserveNodePackageReportTransmissions(nodeName string, sendingType InventoryTransmission, scannerVersion string) {
	nodePackageReportTransmissions.With(prometheus.Labels{
		"node_name":         nodeName,
		"transmission_type": string(sendingType),
		"scanner_version":   scannerVersion,
	}).Inc()
	// Record the old metric for backwards compatibility
	// Remove the block below after Scanner v2 is out of support for clusters
	inventoryTransmissions.With(prometheus.Labels{
		"node_name":         nodeName,
		"transmission_type": string(sendingType),
		"scanner_version":   scannerVersion,
	}).Inc()
}

func init() {
	prometheus.MustRegister(
		callToNodeInventoryDuration,
		inventoryTransmissions,
		nodePackageReportTransmissions,
		numberOfRHELPackages,
		numberOfReportPackages,
		numberOfContentSets,
		numberOfReportContentSets,
		protobufMessageSize,
		protobufReportMessageSize,
		rescanInterval,
		scanDuration,
		indexDuration,
		scansTotal,
		indexesTotal)
}
