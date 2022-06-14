package metrics

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/stackrox/pkg/metrics"
)

var (
	timeSpentInExponentialBackoff = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "enricher_image_scan_internal_exponential_backoff_seconds",
		Help:      "Time spent in exponential backoff for the ImageScanInternal endpoint",
		Buckets:   prometheus.ExponentialBuckets(4, 2, 8),
	})
	networkPoliciesStored = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "num_network_policies_in_store",
		Help:      "Number of network policies (per namespace) currently stored in the sensor's memory.",
	},
		[]string{
			// Which namespace the network policy belongs to
			"k8sNamespace",
		})
	networkPoliciesStoreEvents = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "events_network_policy_store_total",
		Help:      "Events affecting the state of network policies currently stored in the sensor's memory.",
	},
		[]string{
			// What event caused an update of the metric value
			"event",
			// Namespace of the network policy that triggered the metric update
			"k8sNamespace",
			// Number of selector terms on the network policy that triggered the metric update
			"numSelectors",
		})
)

// ObserveTimeSpentInExponentialBackoff observes the metric.
func ObserveTimeSpentInExponentialBackoff(t time.Duration) {
	timeSpentInExponentialBackoff.Observe(t.Seconds())
}

// ObserveNetworkPolicyStoreState observes the metric.
func ObserveNetworkPolicyStoreState(ns string, num int) {
	networkPoliciesStored.With(prometheus.Labels{"k8sNamespace": ns}).Set(float64(num))
}

// ObserveNetworkPolicyStoreEvent observes the metric.
func ObserveNetworkPolicyStoreEvent(event, namespace string, numSelectors int) {
	networkPoliciesStoreEvents.With(prometheus.Labels{
		"event":        event,
		"k8sNamespace": namespace,
		"numSelectors": fmt.Sprintf("%d", numSelectors),
	}).Inc()
}

func init() {
	prometheus.MustRegister(timeSpentInExponentialBackoff, networkPoliciesStored, networkPoliciesStoreEvents)
}
