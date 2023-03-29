package postgres

import (
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/stringutils"
)

func init() {
	prometheus.MustRegister(
		queryDuration,
		queryErrors,
	)
}

var (
	queryDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "postgres_query_duration",
		Help:      "Time in ms for a query to execute",
		Buckets:   prometheus.ExponentialBuckets(4, 2, 16),
	}, []string{"scope", "query"})

	queryErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "postgres_query_errors",
		Help:      "Counter of errors occurring Postgres",
	}, []string{"query", "error"})
)

func setQueryDuration(t time.Time, scope, query string) {
	if strings.HasPrefix(query, "FETCH") {
		query = stringutils.GetUpTo(query, "_")
	}
	query = strings.ReplaceAll(query, "\n", " ")

	logSlowQuery(t, query)

	queryDuration.With(prometheus.Labels{"scope": scope, "query": query}).Observe(float64(time.Since(t).Milliseconds()))
}

func incQueryErrors(query string, err error) {
	if err == pgx.ErrNoRows {
		return
	}
	queryErrors.With(prometheus.Labels{"query": query, "error": err.Error()}).Inc()
}
