package scheduler

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(reportJobsFired)
}

const coPrefix = "complianceoperator_"

var reportJobsFired = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: metrics.PrometheusNamespace,
	Subsystem: metrics.CentralSubsystem.String(),
	Name:      coPrefix + "report_jobs_fired_total",
	Help:      "Compliance report jobs fired by the report scheduler, by scan configuration name",
}, []string{"name"})
