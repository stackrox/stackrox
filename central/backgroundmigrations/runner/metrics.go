package runner

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	bgMigrationSeqNumGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "background_migration_seq_num",
		Help:      "Current sequence number of completed background migrations",
	})

	bgMigrationCompleteGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "background_migration_complete",
		Help:      "1 if all background migrations have finished, 0 otherwise",
	})
)

func init() {
	prometheus.MustRegister(
		bgMigrationSeqNumGauge,
		bgMigrationCompleteGauge,
	)
}
