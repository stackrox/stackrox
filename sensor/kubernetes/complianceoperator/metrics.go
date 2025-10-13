package complianceoperator

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(
		commandsFromCentral,
		applyScanConfigCommands,
	)
}

const (
	coPrefix = "complianceoperator_"
)

var (
	commandsFromCentral = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      coPrefix + "commands_from_central_total",
		Help:      "Total number of messages from Central instructing Sensor to execute a compliance operator related operation",
	}, []string{"operation", "processed"})
	applyScanConfigCommands = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      coPrefix + "apply_scan_config_commands_from_central_total",
		Help:      "Total number of messages from Central instructing Sensor to apply a particular scan config",
	}, []string{"operation"})
)
