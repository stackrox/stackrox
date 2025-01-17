package postgres

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	sensorEventsDeduperCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "image_upsert_deduper",
		Help:      "A counter that tracks the number of deduped image upserts",
	}, []string{"status"})
)

func init() {
	prometheus.MustRegister(sensorEventsDeduperCounter)
}
