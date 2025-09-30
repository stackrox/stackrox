package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
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

// IndexReportsNotRelayed is a counter for the number of virtual machine index reports that failed to get relayed to
// sensor for various reasons.
var IndexReportsNotRelayed = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_index_reports_not_relayed_total",
		Help:      "Total number of virtual machine index reports failed to get relayed to sensor by this Relay",
	},
)

// IndexReportsRelayed is a counter for the number of virtual machine index reports relayed to sensor.
var IndexReportsRelayed = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_index_reports_relayed_total",
		Help:      "Total number of virtual machine index reports relayed to sensor by this Relay",
	},
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
		IndexReportsNotRelayed,
		IndexReportsRelayed,
	)
}
