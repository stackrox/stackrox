package telemetry

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	configDS "github.com/stackrox/rox/central/config/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance *vulnerabilityMetricsImpl
)

func Singleton() interface {
	Start()
	Stop()
} {
	once.Do(func() {

		systemPrivateConfig, err := configDS.Singleton().GetPrivateConfig(context.Background())
		if err != nil {
			logging.LoggerForModule().Errorw("Failed to read Prometheus metrics configuration", logging.Err(err))
			return
		}

		metricsConfig := systemPrivateConfig.GetPrometheusMetricsConfig()

		instance = &vulnerabilityMetricsImpl{
			ds:        deploymentDS.Singleton(),
			metrics:   parseConfig(metricsConfig),
			period:    time.Hour * time.Duration(metricsConfig.GetGatheringPeriodHours()),
			trackFunc: metrics.SetAggregatedVulnCount,
		}
		if instance.period == 0 {
			return
		}
		for metric, expressions := range instance.metrics {
			metrics.RegisterVulnAggregatedMetric(metric, instance.period,
				getMetricLabels(expressions), Problemetrics)
		}
	})
	return instance
}

func parseConfig(config *storage.PrometheusMetricsConfig) map[metricName][]*expression {
	result := make(map[metricName][]*expression)
	for _, metric := range config.GetMetrics() {
		for _, label := range metric.GetLabels() {
			result[metric.GetName()] = append(result[metric.GetName()],
				&expression{
					label: label.GetName(),
					op:    label.GetExpression().Operator,
					arg:   label.GetExpression().Argument,
				})
		}
	}
	return result
}

type metricName = string
type metricKey = string // e.g. IMPORTANT_VULNERABILITY_SEVERITY|true

type vulnerabilityMetricsImpl struct {
	ds         deploymentDS.DataStore
	stopSignal chan bool

	period    time.Duration
	metrics   map[metricName][]*expression
	trackFunc func(metricName string, labels map[Label]string, total int)
}

func (h *vulnerabilityMetricsImpl) Start() {
	if h != nil {
		go h.run()
	}
}

func (h *vulnerabilityMetricsImpl) Stop() {
	if h != nil {
		close(h.stopSignal)
	}
}

func (h *vulnerabilityMetricsImpl) run() {
	ticker := time.NewTicker(h.period)
	defer ticker.Stop()
	ctx, cancel := context.WithCancel(
		sac.WithAllAccess(context.Background()))
	h.track(ctx)
	defer cancel()
	for {
		select {
		case <-ticker.C:
			h.track(ctx)
		case <-h.stopSignal:
			return
		}
	}
}

func (h *vulnerabilityMetricsImpl) track(ctx context.Context) {
	for metric, records := range h.trackVulnerabilityMetrics(ctx) {
		for _, rec := range records {
			h.trackFunc(metric, rec.labels, rec.total)
		}
	}
}

var Problemetrics = prometheus.NewRegistry()
