package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(
		PostgresTableCounts,
		PostgresIndexSize,
		PostgresTableTotalSize,
		PostgresTableDataSize,
		PostgresToastSize,
		PostgresDBSize,
		PostgresTotalSize,
		PostgresRemainingCapacity,
		PostgresConnected,
		PostgresTotalConnections,
		PostgresMaximumConnections,
	)
}

// These variables are all of the stats for Postgres
var (
	PostgresTableCounts = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "postgres_table_size",
		Help:      "estimated number of rows in the table",
	}, []string{"table"})

	PostgresIndexSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "postgres_table_index_bytes",
		Help:      "bytes being used by indexes for a table",
	}, []string{"table"})

	PostgresTableTotalSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "postgres_table_total_bytes",
		Help:      "bytes being used by the table overall",
	}, []string{"table"})

	PostgresTableDataSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "postgres_table_data_bytes",
		Help:      "bytes being used by the data for a table",
	}, []string{"table"})

	PostgresToastSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "postgres_table_toast_bytes",
		Help:      "bytes being used by toast for a table",
	}, []string{"table"})

	PostgresDBSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "postgres_db_size_bytes",
		Help:      "bytes being used by a Postgres Database",
	}, []string{"database"})

	PostgresTotalSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "postgres_total_size_bytes",
		Help:      "bytes being used by Postgres all Databases",
	})

	PostgresRemainingCapacity = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "postgres_available_size_bytes",
		Help:      "remaining bytes available for Postgres",
	})

	PostgresConnected = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "postgres_connected",
		Help:      "flag indicating if central is connected to the Postgres Database. 0 NOT connected, 1 connected",
	})

	PostgresTotalConnections = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "postgres_total_connections",
		Help:      "number of total connections to Postgres by state",
	}, []string{"state"})

	PostgresMaximumConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "postgres_maximum_db_connections",
		Help:      "number of total connections allowed to the Postgres database server",
	})
)
