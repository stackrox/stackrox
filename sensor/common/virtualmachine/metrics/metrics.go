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

// VirtualMachineReceived is a counter for the number of virtual machines received.
var VirtualMachineReceived = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "virtual_machines_received_total",
		Help:      "Total number of virtual machines received by this Sensor",
	},
)

// VirtualMachineSent is a counter for the number of virtual machines sent.
var VirtualMachineSent = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "virtual_machines_sent_total",
		Help:      "Total number of virtual machines sent by this Sensor",
	},
	[]string{"status"},
)

func init() {
	prometheus.MustRegister(
		VirtualMachineReceived,
		VirtualMachineSent,
	)
}
