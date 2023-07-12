package postgres

import (
	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(
		queryErrors,
	)
}

var (
	queryErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "postgres_query_errors",
		Help:      "Counter of errors occurring Postgres",
	}, []string{"query", "error"})
)

func incQueryErrors(query string, err error) {
	if err == nil || err == pgx.ErrNoRows {
		return
	}
	queryErrors.With(prometheus.Labels{"query": query, "error": err.Error()}).Inc()
}
