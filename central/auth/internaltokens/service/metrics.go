package service

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	tokenGenerationDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "internal_token_generation_duration_seconds",
		Help:      "Duration of internal token generation requests",
		Buckets:   prometheus.DefBuckets,
	})
)

func init() {
	prometheus.MustRegister(tokenGenerationDuration)
}
