package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

// IndexReportsMismatchingVsockCID is a counter for the number of virtual machine index reports whose reported vsock CID does not
// match the connection's vsock CID.
var IndexReportsMismatchingVsockCID = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_index_reports_mismatching_vsock_cid_total",
		Help:      "Total number of virtual machine index reports received by this Relay with mismatching vsock CID",
	},
)

// IndexReportsReceived is a counter for the number of virtual machine index reports received.
var IndexReportsReceived = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_index_reports_received_total",
		Help:      "Total number of virtual machine index reports received by this Relay",
	},
)

// IndexReportsSentToSensor is a counter for the number of virtual machine index reports sent to sensor.
var IndexReportsSentToSensor = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_index_reports_sent_total",
		Help:      "Total number of virtual machine index reports sent to sensor by this Relay",
	},
	[]string{"failed"},
)

// ConnectionsAccepted is a counter for the number of connections accepted by this relay. A mismatch between
// this and IndexReportsReceived indicates issues reading or parsing data.
var ConnectionsAccepted = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_connections_accepted_total",
		Help:      "Total number of connections accepted by this Relay",
	},
)

// SemaphoreAcquisitionFailures is a counter for the number of times the connection-handling semaphore that limits
// concurrency could not be acquired. A likely and significant reason for that is that the maximum parallel connections
// were reached.
var SemaphoreAcquisitionFailures = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_sem_acquisition_failures_total",
		Help:      "Number of failed attempts to acquire connection-handling semaphore",
	},
	[]string{"reason"},
)

var SemaphoreHoldingSize = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_sem_holding_size",
		Help:      "Number of connections being handled",
	})

var SemaphoreQueueSize = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_sem_queue_size",
		Help:      "Number of connections waiting to be handled",
	})

// VMIndexReportSendAttempts counts send attempts to Sensor by result.
var VMIndexReportSendAttempts = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_index_report_send_attempts_total",
		Help:      "Send attempts of VM index reports to Sensor partitioned by result",
	},
	[]string{"result"}, // success|retry
)

// VMIndexReportSendDurationSeconds observes per-attempt latency to Sensor by result.
var VMIndexReportSendDurationSeconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_index_report_send_duration_seconds",
		Help:      "Duration of VM index report send attempts to Sensor",
		Buckets:   prometheus.ExponentialBuckets(0.1, 2, 10),
	},
	[]string{"result"}, // success|retry
)

// ReportsRateLimited counts reports dropped by relay-side rate limiting.
var ReportsRateLimited = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_reports_rate_limited_total",
		Help:      "Reports dropped due to relay-side rate limiting",
	},
	[]string{"reason"}, // "normal", "stale_ack"
)

// AcksReceived counts ACK confirmations received from Sensor for VM index reports.
// NACKs are tracked separately in the main compliance component where they're handled.
var AcksReceived = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_acks_received_total",
		Help:      "ACK confirmations received from Sensor for VM index reports",
	},
)

// VMIndexACKsFromSensor counts ACK/NACK responses received from Sensor for VM index reports.
// This metric is recorded when compliance.go handles ComplianceACK messages.
var VMIndexACKsFromSensor = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_index_acks_from_sensor_total",
		Help:      "ACK/NACK responses received from Sensor for VM index reports",
	},
	[]string{"action"}, // "ACK", "NACK"
)

func init() {
	prometheus.MustRegister(
		IndexReportsMismatchingVsockCID,
		IndexReportsReceived,
		IndexReportsSentToSensor,
		ConnectionsAccepted,
		SemaphoreAcquisitionFailures,
		SemaphoreHoldingSize,
		SemaphoreQueueSize,
		VMIndexReportSendAttempts,
		VMIndexReportSendDurationSeconds,
		ReportsRateLimited,
		AcksReceived,
		VMIndexACKsFromSensor,
	)
}
