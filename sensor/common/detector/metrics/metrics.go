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
	receivedNodeInventory = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "node_inventories_received_total",
		Help:      "Total number of Node Inventories received by this sensor",
	},
		[]string{
			// Name of the node sending an inventory
			"node_name",
		})
	receivedNodeInventoryAck = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "node_inventory_ack_received_total",
		Help:      "Total number of Acks or Nacks for Node Inventories received by this sensor",
	},
		[]string{
			// Name of the node sending an inventory
			"node_name",
			"origin",
			"ack_type",
			"reason",
		})

	// DetectorProcessIndicatorBufferSize keeps track of the size of the detection process indicator buffer.
	DetectorProcessIndicatorBufferSize = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "detector_process_indicator_buffer_size",
		Help:      "A counter that tracks the size of the detection process indicator buffer",
	}, []string{"Operation"})

	// DetectorNetworkFlowBufferSize keeps track of the size of the detection network flow buffer.
	DetectorNetworkFlowBufferSize = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "detector_network_flow_buffer_size",
		Help:      "A counter that tracks the size of the detection network flow buffer",
	}, []string{"Operation"})

	// DetectorProcessIndicatorDroppedCount keeps track of the number of process indicators dropped in the detector.
	DetectorProcessIndicatorDroppedCount = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "detector_process_indicator_dropper_counter",
		Help:      "A counter of the total number of process indicators that were dropped if the detector buffer was full",
	})

	// DetectorNetworkFlowDroppedCount keeps track of the number of network flows dropped in the detector.
	DetectorNetworkFlowDroppedCount = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "detector_network_flow_dropper_counter",
		Help:      "A counter of the total number of network flows that were dropped if the detector buffer was full",
	})
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

// ObserveReceivedNodeInventory observes the metric.
func ObserveReceivedNodeInventory(inventory *storage.NodeInventory) {
	receivedNodeInventory.With(prometheus.Labels{
		"node_name": inventory.GetNodeName(),
	}).Inc()
}

// ObserveNodeInventoryAck records (in Sensor) the instance of Central sending (N)Ack to Sensor
func ObserveNodeInventoryAck(nodeName, ackType string, reason AckReason, origin AckOrigin) {
	receivedNodeInventoryAck.With(prometheus.Labels{
		"node_name": nodeName,
		"origin":    string(origin),
		"ack_type":  ackType,
		"reason":    string(reason),
	}).Inc()
}

func init() {
	prometheus.MustRegister(timeSpentInExponentialBackoff,
		networkPoliciesStored,
		networkPoliciesStoreEvents,
		receivedNodeInventory,
		DetectorNetworkFlowBufferSize,
		DetectorProcessIndicatorBufferSize,
		DetectorNetworkFlowDroppedCount,
		DetectorProcessIndicatorDroppedCount)
}
