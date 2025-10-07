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

// VsockConnectionsAccepted is a counter for the number of vsock connections accepted by this relay. A mismatch between
// this and IndexReportsReceived indicates issues reading or parsing data
var VsockConnectionsAccepted = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_vsock_connections_accepted_total",
		Help:      "Total number of vsock connections accepted by this Relay",
	},
)

func init() {
	prometheus.MustRegister(
		IndexReportsMismatchingVsockCID,
		IndexReportsReceived,
		IndexReportsSentToSensor,
		VsockConnectionsAccepted,
	)
}
