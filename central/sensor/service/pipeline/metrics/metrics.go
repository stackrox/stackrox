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
		Name:      "deduped_deployment_count",
		Help:      "Count of the number of deployments that were deduped in their entirety",
	})

	alertsReceivedCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "alerts_received_count",
		Help:      "Count of the number of individual alerts by action and lifecycle stage",
	}, []string{"Action", "Lifecycle"})
)

// AddTotalAlertsReceived breaks down AlertResults into a count of individual alerts by action and lifecycle
func AddTotalAlertsReceived(action central.ResourceAction, stage storage.LifecycleStage, alertCount int) {
	alertsReceivedCount.With(
		prometheus.Labels{
			"Action":    action.String(),
			"Lifecycle": stage.String(),
		}).Add(float64(alertCount))
}

func init() {
	prometheus.MustRegister(
		DedupedDeploymentsCount,
		alertsReceivedCount,
	)
}
