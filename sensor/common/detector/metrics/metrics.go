package metrics

import (
	"fmt"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/centralcaps"
)

// Image origin categories for indexing route metrics.
//
// cluster_local: images from registries that belong to the secured cluster
// itself (e.g. the OpenShift integrated registry). When a local scanner is
// present these images are always scanned locally — no Central round-trip.
//
// non_cluster_local: all other images (Docker Hub, Quay, ECR, GCR, …).
// Where these are indexed depends on the delegated scanning configuration
// pushed by Central (see DelegatedRegistryConfig).
const (
	ForImagesClusterLocal    = "cluster_local"
	ForImagesNonClusterLocal = "non_cluster_local"
)

// Indexer labels describe which component performs image indexing (pulling
// layers and extracting package metadata). Vulnerability matching always
// happens in Central regardless of the indexer.
//
//   - local_scanner:                       the Scanner pod co-deployed with Sensor
//   - central_scanner:                     the Scanner in the Central services cluster
//   - local_scanner_or_central_scanner:    mixed — some registries go local,
//     the rest go to Central (delegated scanning with SPECIFIC registries)
const (
	IndexerLocalScanner   = "local_scanner"
	IndexerCentralScanner = "central_scanner"
	IndexerMixed          = "local_scanner_or_central_scanner"
)

// Scanner mode labels indicate which scanner generation the local scanner
// would use. Determined at runtime by the ROX_SCANNER_V4 feature flag AND
// whether Central advertises the ScannerV4Supported capability.
// The mode does NOT verify that the scanner pod is actually running.
const (
	ModeNone = "none"
	ModeV2   = "v2"
	ModeV4   = "v4"
)

// AckOrigin tells what entity issued the Ack message
type AckOrigin string

const (
	// AckOriginUnknown is default value and should be used when origin of the ack is unknown
	AckOriginUnknown AckOrigin = "Unknown"
	// AckOriginCentral marks Central as the entity that produced the ack
	AckOriginCentral AckOrigin = "Central"
	// AckOriginSensor marks Sensor as the entity that produced the ack
	AckOriginSensor AckOrigin = "Sensor"
)

// AckReason tells why a given ACK was sent (used mainly for NACKs)
type AckReason string

const (
	// AckReasonUnknown is default value and should be used when reason for emitting the ack is unknown
	AckReasonUnknown AckReason = "Unknown reason"
	// AckReasonNodeUnknown is used by Sensor when node inventory refers to a K8s node that is not known yet to sensor
	AckReasonNodeUnknown AckReason = "Node unknown to Sensor"
	// AckReasonCentralUnreachable is used by Sensor when node inventory cannot be sent to Central
	AckReasonCentralUnreachable AckReason = "Central unreachable"
	// AckReasonForwardingFromCentral is used for ACKs that arrived from Central and are forwarded to Compliance.
	AckReasonForwardingFromCentral AckReason = "Forwarding from Central"
)

type AckOperation string

const (
	// AckOperationReceive marks receiving ACK from Central
	AckOperationReceive AckOperation = "receive from Central"
	// AckOperationCreate marks creating a new ACK in Sensor
	AckOperationCreate AckOperation = "create in Sensor"
	// AckOperationSend marks sending ACK scan to Compliance
	AckOperationSend AckOperation = "send to Compliance"
)

// NodeScanOperation denotes operations done on Node Invetory and Node Scan within Sensor
type NodeScanOperation string

const (
	// NodeScanOperationReceive marks receiving node scan from Compliance
	NodeScanOperationReceive NodeScanOperation = "receive from Compliance"
	// NodeScanOperationSendToCentral marks sending node scan to Central
	NodeScanOperationSendToCentral NodeScanOperation = "send to Central"
)

// NodeScanType denotes whether the object is NodeInventory or NodeIndex
type NodeScanType string

// Keeping the const name in sync with MsgToCompliance_NodeInventoryACK for easier metrics analysis.
const (
	// NodeScanTypeNodeInventory represents NodeInventory
	NodeScanTypeNodeInventory NodeScanType = "NodeInventory"
	// NodeScanTypeNodeIndex represents NodeIndex
	NodeScanTypeNodeIndex NodeScanType = "NodeIndexer"
)

var (
	timeSpentInExponentialBackoff = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "enricher_image_scan_internal_exponential_backoff_seconds",
		Help:      "Time spent backing off before a successful ImageScanInternal response, typically due to scan rate limiting",
		Buckets:   prometheus.ExponentialBuckets(1, 2, 10),
	})
	networkPoliciesStored = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "num_network_policies_in_store",
		Help:      "Number of network policies (per namespace) currently stored in the sensor's memory.",
	},
		[]string{
			// Which namespace the network policy belongs to
			"k8sNamespace",
		})
	networkPoliciesStoreEvents = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "events_network_policy_store_total",
		Help:      "Events affecting the state of network policies currently stored in the sensor's memory.",
	},
		[]string{
			// What event caused an update of the metric value
			"event",
			// Namespace of the network policy that triggered the metric update
			"k8sNamespace",
			// Number of selector terms on the network policy that triggered the metric update
			"numSelectors",
		})
	// processedNodeScan is a metric meant to replace and provide extra context on
	// receivedNodeInventory and receivedNodeIndex
	processedNodeScan = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "node_scan_processed_total",
		Help:      "Counts node inventory/index reports received from Compliance and sent to Central",
	},
		[]string{
			// Name of the node sending an inventory
			"node_name",
			"type",
			"operation",
		})
	receivedNodeInventory = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "node_inventories_received_total",
		Help:      "Total number of Node Inventories received by this Sensor",
	},
		[]string{
			// Name of the node sending an inventory
			"node_name",
		})
	receivedNodeIndex = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "node_indexes_received_total",
		Help:      "Total number of Node Indexes received by this Sensor",
	},
		[]string{
			// Name of the node sending an inventory
			"node_name",
		})
	processedNodeScanningAck = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "node_scanning_ack_processed_total",
		Help:      "Counts ACK/NACK messages for node inventory/index processing",
	},
		[]string{
			// Name of the node sending an inventory
			"node_name",
			"origin",
			"ack_type",
			"operation",
			"message_type",
			"reason",
		})

	// FileAccessEventsReceived counts file access events entering the pipeline.
	FileAccessEventsReceived = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "file_access_events_received_total",
		Help:      "Total number of file access events received from the file activity monitoring agent and entering the processing pipeline",
	})

	// FileAccessCriteriaMatchDuration tracks how long criteria matching takes per file access event.
	FileAccessCriteriaMatchDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "file_access_criteria_match_duration_seconds",
		Help:      "Time spent matching file access criteria per event",
		Buckets:   prometheus.DefBuckets,
	})

	// DetectorProcessIndicatorQueueOperations keeps track of the operations of the detection process indicator buffer.
	DetectorProcessIndicatorQueueOperations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "detector_process_indicator_queue_operations_total",
		Help:      "A counter that tracks the number of ADD and REMOVE operations on the process indicator buffer queue. Current size of the queue can be calculated by subtracting the number of remove operations from the add operations",
	}, []string{"Operation"})

	// DetectorNetworkFlowQueueOperations keeps track of the operations of the detection network flow buffer.
	DetectorNetworkFlowQueueOperations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "detector_network_flow_queue_operations_total",
		Help:      "A counter that tracks the number of ADD and REMOVE operations on the network flows buffer queue. Current size of the queue can be calculated by subtracting the number of remove operations from the add operations",
	}, []string{"Operation"})

	// DetectorDeploymentQueueOperations keeps track of the operations of the detection deployment buffer.
	DetectorDeploymentQueueOperations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "detector_deployment_queue_operations_total",
		Help:      "A counter that tracks the number of ADD and REMOVE operations on the deployment buffer queue. Current size of the queue can be calculated by subtracting the number of remove operations from the add operations",
	}, []string{"Operation"})

	// DetectorFileAccessQueueOperations keeps track of the operations of the
	// detection file access buffer.
	DetectorFileAccessQueueOperations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "detector_file_access_queue_operations_total",
		Help:      "A counter that tracks the number of ADD and REMOVE operations on the file access buffer queue. Current size of the queue can be calculated by subtracting the number of remove operations from the add operations",
	}, []string{"Operation"})

	// DetectorProcessIndicatorDroppedCount keeps track of the number of process indicators dropped in the detector.
	DetectorProcessIndicatorDroppedCount = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "detector_process_indicator_queue_dropped_total",
		Help:      "A counter of the total number of process indicators that were dropped if the detector buffer was full",
	})

	// DetectorNetworkFlowDroppedCount keeps track of the number of network flows dropped in the detector.
	DetectorNetworkFlowDroppedCount = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "detector_network_flow_queue_dropped_total",
		Help:      "A counter of the total number of network flows that were dropped if the detector buffer was full",
	})

	// DetectorDeploymentDroppedCount keeps track of the number of deployments dropped in the detector.
	DetectorDeploymentDroppedCount = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "detector_deployment_queue_dropped_total",
		Help:      "A counter of the total number of deployments that were dropped if the detector buffer was full",
	})

	// DetectorFileAccessDroppedCount keeps track of the number of file accesses dropped in the detector.
	DetectorFileAccessDroppedCount = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "detector_file_access_queue_dropped_total",
		Help:      "A counter of the total number of file accesses that were dropped if the detector buffer was full",
	})

	detectorBlockScanCalls = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "block_scan_calls_total",
		Help:      "Counts add/remove operations for blocking scans triggered by deployment create/update",
	}, []string{"Operation", "Path"})

	scanCallDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "scan_call_duration_milliseconds",
		Help: "Total time spent calling Scan in milliseconds, including retries and backoff waits. " +
			"Applies to both local and remote scans (whichever is currently used in Sensor).",
		Buckets: prometheus.ExponentialBuckets(4, 2, 16),
	})

	scanAndSetCall = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "scan_and_set_calls_total",
		Help:      "A counter that tracks the operations in scan and set",
	}, []string{"Operation", "Reason"})

	// Scanner topology truth table — every valid combination of labels.
	//
	// Scanning has three layered concerns:
	//   1. Local scanner presence (ROX_LOCAL_IMAGE_SCANNING_ENABLED).
	//      When true a Scanner pod runs next to Sensor and can index images
	//      from the cluster's own registries without a Central round-trip.
	//   2. Scanner generation (mode): v4 when ROX_SCANNER_V4=true AND Central
	//      advertises ScannerV4Supported; otherwise v2. Does NOT verify that
	//      the scanner pod is actually running.
	//   3. Delegated scanning: Central pushes a DelegatedRegistryConfig that
	//      tells Sensor whether non-cluster-local images should also be indexed
	//      locally. Requires local=true AND ROX_DELEGATED_SCANNING_DISABLED=false.
	//
	//  local | mode | delegated | cluster-local images | non-cluster-local images
	//  ------+------+-----------+----------------------+--------------------------
	//  false | none | false     | Central scanner      | Central scanner
	//  true  | v2   | false     | local scanner (v2)   | Central scanner
	//  true  | v4   | false     | local scanner (v4)   | Central scanner
	//  true  | v2   | true/ALL  | local scanner (v2)   | local scanner (v2)
	//  true  | v4   | true/ALL  | local scanner (v4)   | local scanner (v4)
	//  true  | v2   | true/SPEC | local scanner (v2)   | local or Central (per registry)
	//  true  | v4   | true/SPEC | local scanner (v4)   | local or Central (per registry)
	//
	// Impossible states (never emitted):
	//   local=false, mode=v2              — no scanner means no mode.
	//   local=false, mode=v4              — no scanner means no mode.
	//   local=true,  mode=none            — a local scanner is always v2 or v4.
	//   local=false, delegated=true       — delegation requires a local scanner.
	//   local=true,  mode=none, delegated=true — compound of the above two.
	//
	// Note: vulnerability matching always happens in Central regardless of
	// where indexing takes place.
	scannerConfigurationInfo = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "scanner_configuration_info",
		Help: "Describes how image scanning is configured on this Sensor. " +
			"local=true means a Scanner pod runs alongside Sensor and handles cluster-local images (e.g. OCP internal registry). " +
			"mode is the scanner generation (v2 or v4; none when local scanning is off). " +
			"delegated=true means Central also routes non-cluster-local images to the local scanner " +
			"(requires local=true, ROX_DELEGATED_SCANNING_DISABLED=false, and Central config EnabledFor != NONE). " +
			"Vulnerability matching always happens in Central.",
	}, []string{"local", "mode", "delegated"})

	// Emits one series per image origin (cluster_local / non_cluster_local),
	// showing which component performs the indexing step (pulling layers and
	// extracting package metadata). This is the *effect* of the scanner
	// configuration above — use scannerConfigurationInfo to see the *inputs*.
	//
	//  for_images          | indexer (when local=false) | indexer (when local=true, delegated off) | indexer (delegated ALL) | indexer (delegated SPECIFIC)
	//  --------------------+---------------------------+-----------------------------------------+------------------------+-----------------------------
	//  cluster_local       | central_scanner           | local_scanner                           | local_scanner           | local_scanner
	//  non_cluster_local   | central_scanner           | central_scanner                         | local_scanner           | local_scanner_or_central_scanner
	imageIndexingRouteInfo = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "image_indexing_route_info",
		Help: "Shows where image indexing happens for each image origin. " +
			"for_images=cluster_local covers images from the cluster's own registries (e.g. OCP internal); " +
			"for_images=non_cluster_local covers everything else (Docker Hub, Quay, ECR, …). " +
			"indexer is local_scanner, central_scanner, or local_scanner_or_central_scanner (mixed, " +
			"when delegated scanning targets SPECIFIC registries). " +
			"Vulnerability matching always happens in Central regardless of where indexing occurs.",
	}, []string{"for_images", "indexer"})
)

// ObserveTimeSpentInExponentialBackoff observes the metric.
func ObserveTimeSpentInExponentialBackoff(t time.Duration) {
	timeSpentInExponentialBackoff.Observe(t.Seconds())
}

// ObserveNetworkPolicyStoreState observes the metric.
func ObserveNetworkPolicyStoreState(ns string, num int) {
	networkPoliciesStored.With(prometheus.Labels{"k8sNamespace": ns}).Set(float64(num))
}

// ObserveNetworkPolicyStoreEvent observes the metric.
func ObserveNetworkPolicyStoreEvent(event, namespace string, numSelectors int) {
	networkPoliciesStoreEvents.With(prometheus.Labels{
		"event":        event,
		"k8sNamespace": namespace,
		"numSelectors": fmt.Sprintf("%d", numSelectors),
	}).Inc()
}

// ObserveNodeScan observes the metric.
func ObserveNodeScan(nodeName string, typ NodeScanType, op NodeScanOperation) {
	processedNodeScan.With(prometheus.Labels{
		"node_name": nodeName,
		"type":      string(typ),
		"operation": string(op),
	}).Inc()
}

// ObserveReceivedNodeInventory observes the metric.
func ObserveReceivedNodeInventory(inventory *storage.NodeInventory) {
	receivedNodeInventory.With(prometheus.Labels{
		"node_name": inventory.GetNodeName(),
	}).Inc()
}

// ObserveReceivedNodeIndex observes the metric.
func ObserveReceivedNodeIndex(nodeName string) {
	receivedNodeIndex.With(prometheus.Labels{
		"node_name": nodeName,
	}).Inc()
}

// ObserveNodeScanningAck records (in Sensor) the instance of Central sending (N)Ack to Sensor
func ObserveNodeScanningAck(nodeName, ackType, messageType string, op AckOperation, reason AckReason, origin AckOrigin) {
	processedNodeScanningAck.With(prometheus.Labels{
		"node_name":    nodeName,
		"origin":       string(origin),
		"ack_type":     ackType,
		"operation":    string(op),
		"message_type": messageType,
		"reason":       string(reason),
	}).Inc()
}

// AddBlockingScanCall records a call to blockingScan
func AddBlockingScanCall(path string) {
	detectorBlockScanCalls.With(prometheus.Labels{
		"Operation": metrics.Add.String(),
		"Path":      path,
	}).Inc()
}

// RemoveBlockingScanCall records a call to blockingScan has finished
func RemoveBlockingScanCall() {
	detectorBlockScanCalls.With(prometheus.Labels{
		"Operation": metrics.Remove.String(),
		"Path":      "",
	}).Inc()
}

// SetScanCallDuration records the duration of the scan call to central/scanner
func SetScanCallDuration(start time.Time) {
	now := time.Now()
	durMilli := now.Sub(start).Milliseconds()
	scanCallDuration.Observe(float64(durMilli))
}

// AddScanAndSetCall records a call to ScanAndSet
func AddScanAndSetCall(reason string) {
	scanAndSetCall.With(prometheus.Labels{
		"Operation": metrics.Add.String(),
		"Reason":    reason,
	}).Inc()
}

// RemoveScanAndSetCall records a call to ScanAndSet has finished
func RemoveScanAndSetCall(reason string) {
	scanAndSetCall.With(prometheus.Labels{
		"Operation": metrics.Remove.String(),
		"Reason":    reason,
	}).Inc()
}

// scannerConfigMu serialises UpdateScannerConfigurationInfo so that the
// multi-step Reset → Set sequence on two gauge vectors is atomic. Individual
// Prometheus operations are thread-safe, but without the lock two concurrent
// callers could interleave (e.g. Reset one vector while the other still has
// stale series).
//
// lastConfig retains the most recent non-nil DelegatedRegistryConfig so that
// capability-only refreshes (the hello handshake passes nil) do not
// temporarily reset delegated-routing metrics between a reconnect and the
// next config delivery from Central.
var (
	scannerConfigMu sync.Mutex
	lastConfig      *central.DelegatedRegistryConfig
)

func resetLastDelegatedConfig() {
	scannerConfigMu.Lock()
	defer scannerConfigMu.Unlock()
	lastConfig = nil
}

// scannerMode determines which scanner generation label to emit.
//
// Decision tree:
//
//	localEnabled=false → "none"  (no local scanner at all)
//	ROX_SCANNER_V4=true AND Central has ScannerV4Supported → "v4"
//	otherwise → "v2"  (legacy scanner or Central hasn't confirmed V4 yet)
func scannerMode(localEnabled bool) string {
	if !localEnabled {
		return ModeNone
	}
	if features.ScannerV4.Enabled() && centralcaps.Has(centralsensor.ScannerV4Supported) {
		return ModeV4
	}
	return ModeV2
}

// isDelegatedEffective answers: "is Sensor actually scanning non-cluster-local
// images locally right now?" This requires ALL of the following:
//
//  1. A local scanner is present (localEnabled=true).
//  2. The kill-switch ROX_DELEGATED_SCANNING_DISABLED is false.
//  3. Central has pushed a DelegatedRegistryConfig with EnabledFor != NONE.
//
// config may be nil (proto GetEnabledFor returns NONE on nil receiver), which
// correctly yields false.
func isDelegatedEffective(localEnabled bool, config *central.DelegatedRegistryConfig) bool {
	envEnabled := localEnabled && !env.DelegatedScanningDisabled.BooleanSetting()
	if !envEnabled {
		return false
	}
	return config.GetEnabledFor() != central.DelegatedRegistryConfig_NONE
}

// nonClusterLocalIndexer returns the indexer label for images that do NOT come
// from the cluster's own registry. The result depends on whether delegated
// scanning is active and, if so, how broadly:
//
//	delegated off        → "central_scanner"                   (Central does all indexing)
//	delegated ALL        → "local_scanner"                     (everything indexed locally)
//	delegated SPECIFIC   → "local_scanner_or_central_scanner"  (per-registry routing)
func nonClusterLocalIndexer(localEnabled bool, config *central.DelegatedRegistryConfig) string {
	if !isDelegatedEffective(localEnabled, config) {
		return IndexerCentralScanner
	}
	switch config.GetEnabledFor() {
	case central.DelegatedRegistryConfig_ALL:
		return IndexerLocalScanner
	case central.DelegatedRegistryConfig_SPECIFIC:
		return IndexerMixed
	default:
		return IndexerCentralScanner
	}
}

// UpdateScannerConfigurationInfo refreshes Prometheus info metrics that
// describe the scanner topology of this Sensor. It is safe for concurrent use.
//
// The topology depends on three inputs:
//
//  1. Local scanner presence: ROX_LOCAL_IMAGE_SCANNING_ENABLED env var.
//     When true, a Scanner pod (v2 or v4) runs beside Sensor and handles
//     images from the cluster's own registries (e.g. OCP internal registry).
//
//  2. Scanner generation: determined at runtime from the ROX_SCANNER_V4
//     feature flag AND whether Central advertises the ScannerV4Supported
//     capability. Falls back to v2 if either condition is not met.
//
//  3. Delegated scanning: Central pushes a DelegatedRegistryConfig telling
//     Sensor whether non-cluster-local images should also be scanned locally.
//     Three delegation modes: NONE (no delegation), ALL (everything local),
//     SPECIFIC (only matching registry paths scanned locally).
//     Requires local scanning enabled AND ROX_DELEGATED_SCANNING_DISABLED=false.
//
// config may be nil on reconnect (hello handshake); in that case the last
// known config is reused so metrics remain stable across reconnects.
//
// Called from two places:
//   - Central hello handshake (nil config) — refreshes mode after capabilities update.
//   - Delegated registry handler (non-nil config) — Central pushed new routing config.
func UpdateScannerConfigurationInfo(config *central.DelegatedRegistryConfig) {
	// Hold the lock for the full update (config swap + gauge Reset/Set) so that
	// concurrent callers cannot interleave and leave mixed or zero series.
	// The individual Prometheus operations are thread-safe, but the sequence
	// Reset → Set across two gauge vectors is not atomic without the lock.
	scannerConfigMu.Lock()
	defer scannerConfigMu.Unlock()

	if config != nil {
		lastConfig = config
	} else {
		config = lastConfig
	}

	localEnabled := env.LocalImageScanningEnabled.BooleanSetting()

	scannerConfigurationInfo.Reset()
	scannerConfigurationInfo.With(prometheus.Labels{
		"local":     strconv.FormatBool(localEnabled),
		"mode":      scannerMode(localEnabled),
		"delegated": strconv.FormatBool(isDelegatedEffective(localEnabled, config)),
	}).Set(1)

	clusterLocalIndexer := IndexerCentralScanner
	if localEnabled {
		clusterLocalIndexer = IndexerLocalScanner
	}

	imageIndexingRouteInfo.Reset()
	imageIndexingRouteInfo.With(prometheus.Labels{
		"for_images": ForImagesClusterLocal,
		"indexer":    clusterLocalIndexer,
	}).Set(1)
	imageIndexingRouteInfo.With(prometheus.Labels{
		"for_images": ForImagesNonClusterLocal,
		"indexer":    nonClusterLocalIndexer(localEnabled, config),
	}).Set(1)
}

func init() {
	prometheus.MustRegister(timeSpentInExponentialBackoff,
		networkPoliciesStored,
		networkPoliciesStoreEvents,
		processedNodeScan,
		receivedNodeInventory,
		receivedNodeIndex,
		processedNodeScanningAck,
		DetectorNetworkFlowQueueOperations,
		DetectorProcessIndicatorQueueOperations,
		DetectorNetworkFlowDroppedCount,
		DetectorProcessIndicatorDroppedCount,
		detectorBlockScanCalls,
		scanCallDuration,
		scanAndSetCall,
		DetectorDeploymentQueueOperations,
		DetectorDeploymentDroppedCount,
		DetectorFileAccessQueueOperations,
		DetectorFileAccessDroppedCount,
		FileAccessEventsReceived,
		FileAccessCriteriaMatchDuration,
		scannerConfigurationInfo,
		imageIndexingRouteInfo,
	)
}
