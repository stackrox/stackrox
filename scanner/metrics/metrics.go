package metrics

import (
	"context"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	metricsSrv           Server
	vulnDBUpdateDuration *prometheus.GaugeVec
)

type Server interface {
	RunForever()
	Stop(context.Context)
	RegisterAdditionalCollector(collector prometheus.Collector) error
}

// Initializes metrics server and returns a cleanup function to be deferred from the caller
func Initialize() func(context.Context) {
	metricsSrv = metrics.NewServer(metrics.ScannerSubsystem, metrics.NewTLSConfigurerFromEnv())
	metricsSrv.RunForever()
	metrics.GatherThrottleMetricsForever(metrics.ScannerSubsystem.String())

	vulnDBUpdateDuration = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ScannerSubsystem.String(),
		Name:      "vuln_db_update_duration_seconds",
		Help:      "Time to query, download, and load a vulnerability database bundle, measured end-to-end from the start of the update cycle",
	}, []string{"database", "initial_load"})

	if err := metricsSrv.RegisterAdditionalCollector(vulnDBUpdateDuration); err != nil {
		slog.Error("failed to register vuln db update duration metric", "reason", err)
	}

	return metricsSrv.Stop
}

func GetVulnDBUpdateDuration() *prometheus.GaugeVec {
	return vulnDBUpdateDuration
}
