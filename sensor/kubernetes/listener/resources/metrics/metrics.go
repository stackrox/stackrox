package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	podsStored = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "num_pods_in_store",
		Help:      "A gauge to track the number of pods in the store",
	},
		[]string{
			"k8sNamespace",
		})
)

// UpdateNumberPodsInStored update number of pods stored
func UpdateNumberPodsInStored(ns string, num int) {
	podsStored.With(prometheus.Labels{"k8sNamespace": ns}).Set(float64(num))
}

func init() {
	prometheus.MustRegister(podsStored)
}
