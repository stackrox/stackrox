package telemetry

import (
	"context"

	"time"

	"github.com/prometheus/client_golang/prometheus"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/metrics"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance *vulnerabilityMetricsImpl

	Problemetrics = prometheus.NewRegistry()
)

func Singleton() interface {
	Start()
	Stop()
} {
	once.Do(func() {
		log := logging.LoggerForModule()
		metricsConfig, period, err := parseConfig()
		if err != nil {
			log.Errorw("Failed to parse Prometheus metrics configuration", logging.Err(err))
			return
		}
		if period == 0 {
			return
		}
		instance = &vulnerabilityMetricsImpl{
			ds:        deploymentDS.Singleton(),
			metrics:   metricsConfig,
			period:    period,
			trackFunc: metrics.SetAggregatedVulnCount,
		}
		for metric, expressions := range instance.metrics {
			metrics.RegisterVulnAggregatedMetric(metric, instance.period,
				getMetricLabels(expressions), Problemetrics)

			log.Infof("Registered Prometheus metric %q", metric)
		}
	})
	return instance
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
