package upgradecontroller

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(upgraderTriggered)
	prometheus.MustRegister(upgraderErrors)
}

var (
	upgraderTriggered = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "upgrader_triggered_total",
		Help:      "Number of times the upgrader was triggered.",
	}, []string{"centralVersion", "sensorVersion", "clusterID", "triggerOrigin", "upgradeType"})
	upgraderErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "upgrader_errors_total",
		Help:      "Number of times the upgrader returned error.",
	}, []string{"centralVersion", "sensorVersion", "clusterID", "error", "upgradeType"})
)

func observeUpgraderTriggered(sensorVersion, origin, clusterID string, process *storage.ClusterUpgradeStatus_UpgradeProcessStatus, upgraderActive bool) {
	if upgraderActive {
		upgraderTriggered.With(prometheus.Labels{
			"centralVersion": process.GetTargetVersion(),
			"sensorVersion":  sensorVersion,
			"clusterID":      clusterID,
			"triggerOrigin":  origin,
			"upgradeType":    process.GetType().String()}).Inc()
	}
}

func observeUpgraderError(sensorVersion, clusterID, err string, process *storage.ClusterUpgradeStatus_UpgradeProcessStatus) {
	if err == "" {
		return
	}
	upgradeType := "unknown"
	centralVersion := "unknown"
	if process != nil {
		upgradeType = process.GetType().String()
		centralVersion = process.GetTargetVersion()
	}
	upgraderErrors.With(prometheus.Labels{
		"centralVersion": centralVersion,
		"sensorVersion":  sensorVersion,
		"clusterID":      clusterID,
		"error":          err,
		"upgradeType":    upgradeType}).Inc()
}
