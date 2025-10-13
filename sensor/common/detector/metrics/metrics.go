package metrics

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/metrics"
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
		Help:      "Time spent in exponential backoff for the ImageScanInternal endpoint",
		Buckets:   prometheus.ExponentialBuckets(4, 2, 8),
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
		Help:      "Total number of Node Inventories/Indexes received/sent by this Sensor",
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
		Help:      "Total number of Acks or Nacks for Node Inventories/Indexes processed by Sensor",
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

	detectorBlockScanCalls = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "block_scan_calls_total",
		Help:      "A counter that tracks the operations in blocking scan calls",
	}, []string{"Operation", "Path"})

	scanCallDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "scan_call_duration_milliseconds",
		Help:      "Time taken to call scan in milliseconds",
		Buckets:   prometheus.ExponentialBuckets(4, 2, 16),
	})

	scanAndSetCall = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "scan_and_set_calls_total",
		Help:      "A counter that tracks the operations in scan and set",
	}, []string{"Operation", "Reason"})
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
	)
}
