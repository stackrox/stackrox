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
		Help:      "Time to load an individual vulnerability database bundle",
	}, []string{"database", "initial_load"})

	if err := prometheus.Register(vulnDBUpdateDuration); err != nil {
		slog.Error("failed to register vuln db update duration metric", "reason", err)
	}

	return metricsSrv.Stop
}

// GetVulnDBUpdateDuration returns the vuln DB update duration metric collector. If the
// collector has not been initialized, it returns nil.
func GetVulnDBUpdateDuration() *prometheus.GaugeVec {
	return vulnDBUpdateDuration
}
