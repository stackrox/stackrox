package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	// DedupedDeploymentsCount indicates how many deployments were deduped
	DedupedDeploymentsCount = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "pipeline_deduped_deployment_count",
		Help:      "Count of the number of deployments that were deduped in their entirety",
	})

	alertsReceivedCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "pipeline_alerts_received_count",
		Help:      "Count of the number of individual alerts by action, lifecycle stage and policy",
	}, []string{"Action", "Lifecycle", "Policy"})
)

// IncTotalAlertsReceived breaks down AlertResults into a count of individual alerts by action, lifecycle, and policy
func IncTotalAlertsReceived(action central.ResourceAction, stage storage.LifecycleStage, policy string) {
	alertsReceivedCount.With(
		prometheus.Labels{
			"Action":    action.String(),
			"Lifecycle": stage.String(),
			"Policy":    policy,
		}).Inc()
}

func init() {
	prometheus.MustRegister(
		DedupedDeploymentsCount,
		alertsReceivedCount,
	)
}
