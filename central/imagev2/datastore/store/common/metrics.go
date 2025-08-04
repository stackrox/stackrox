package common

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	SensorEventsDeduperCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "imageV2_upsert_deduper",
		Help:      "A counter that tracks the number of deduped imageV2 upserts",
	}, []string{"status"})
)

func init() {
	prometheus.MustRegister(SensorEventsDeduperCounter)
}
