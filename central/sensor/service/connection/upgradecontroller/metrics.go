package upgradecontroller

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/storage"
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
	}, []string{"centralVersion", "sensorVersion", "clusterID", "triggerOrigin", "upgradeType"})
)

func registerUpgraderTriggered(sensorVersion, origin, clusterID string, process *storage.ClusterUpgradeStatus_UpgradeProcessStatus, upgraderActive bool) {
	if upgraderActive {
		upgraderTriggered.With(prometheus.Labels{
			"centralVersion": process.GetTargetVersion(),
			"sensorVersion":  sensorVersion,
			"clusterID":      clusterID,
			"triggerOrigin":  origin,
			"upgradeType":    process.GetType().String()}).Inc()
	}
}
