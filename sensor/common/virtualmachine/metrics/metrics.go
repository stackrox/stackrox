package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	StatusCentralNotReadyLabels = prometheus.Labels{"status": "central not ready"}
	StatusErrorLabels           = prometheus.Labels{"status": "error"}
	StatusSuccessLabels         = prometheus.Labels{"status": "success"}
	StatusTimeoutLabels         = prometheus.Labels{"status": "timeout"}
)

// IndexReportsReceived is a counter for the number of virtual machine index reports received.
var IndexReportsReceived = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "virtual_machine_index_reports_received_total",
		Help:      "Total number of virtual machine index reports received by this Sensor",
	},
)

// IndexReportsSent is a counter for the number of virtual machine index reports sent.
var IndexReportsSent = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "virtual_machine_index_reports_sent_total",
		Help:      "Total number of virtual machine index reports sent by this Sensor",
	},
	[]string{"status"},
)

func init() {
	prometheus.MustRegister(
		IndexReportsReceived,
		IndexReportsSent,
	)
}
