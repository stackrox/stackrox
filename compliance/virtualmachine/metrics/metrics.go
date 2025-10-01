package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	StatusFailLabels    = prometheus.Labels{"status": "fail"}
	StatusSuccessLabels = prometheus.Labels{"status": "success"}
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
	[]string{"status"},
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
		IndexReportsReceived,
		IndexReportsSentToSensor,
		VsockConnectionsAccepted,
	)
}
