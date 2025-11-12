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

func init() {
	prometheus.MustRegister(
		IndexReportsMismatchingVsockCID,
		IndexReportsReceived,
		IndexReportsSentToSensor,
		ConnectionsAccepted,
		SemaphoreAcquisitionFailures,
		SemaphoreHoldingSize,
		SemaphoreQueueSize,
	)
}
