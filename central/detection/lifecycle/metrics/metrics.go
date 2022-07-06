package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(
		processFilterCounter,
		dedupedAlertsCount,
	)
}

var (
	processFilterCounter = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "process_filter",
		Help:      "Process filter hits and misses",
	}, []string{"Type"})

	dedupedAlertsCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "alert_manager_deduped_alerts_count",
		Help:      "Count number of individual alerts deduped by the manager",
	}, []string{"Lifecycle", "Policy"})
)

// ProcessFilterCounterInc increments a counter for determining how effective the process filter is
func ProcessFilterCounterInc(typ string) {
	processFilterCounter.With(prometheus.Labels{"Type": typ}).Inc()
}

// IncDedupedAlerts increments the number of deduped alerts in the manager
func IncDedupedAlerts(stage storage.LifecycleStage, policy string) {
	dedupedAlertsCount.With(prometheus.Labels{"Lifecycle": stage.String(), "Policy": policy}).Inc()
}
