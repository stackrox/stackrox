package upgradecontroller

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(upgraderTriggered)
}

var (
	upgraderTriggered = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "upgrader_triggered_total",
		Help:      "Number of times the upgrader was triggered.",
	}, []string{"centralVersion", "sensorVersion", "triggerOrigin", "upgradeType", "triggerSucceeded"})
)
