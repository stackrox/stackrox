package telemetry

import (
	"context"
	"sync"
	"time"

	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/pkg/env"
)

var (
	once     sync.Once
	instance *vulnerabilityMetricsImpl
)

func Singleton() *vulnerabilityMetricsImpl {
	once.Do(func() {
		instance = &vulnerabilityMetricsImpl{
			ds:                deploymentDS.Singleton(),
			metricExpressions: parseAggregationExpressions(env.AggregateCVSSMetrics.Setting()),
			trackFunc:         metrics.SetAggregatedVulnCount,
		}
		for metricName, expressions := range instance.metricExpressions {
			labels := getMetricLabels(expressions)
			metrics.RegisterVulnAggregatedMetric(metricName, labels)
		}
	})
	return instance
}

type metricName = string
type metricKey = string // e.g. IMPORTANT_VULNERABILITY_SEVERITY|true

type vulnerabilityMetricsImpl struct {
	ds         deploymentDS.DataStore
	stopSignal chan bool

	// metricExpressions associates a metric name to the list of expressions.
	//
	// Example:
	//
	//   "Namespace_eq_abc_Severity_total": {"Namespace=abc", "Severity"},
	metricExpressions map[metricName][]expression
	trackFunc         func(metricName string, labels map[string]string, total int)
}

func (h *vulnerabilityMetricsImpl) Start() {
	go h.run()
}

func (h *vulnerabilityMetricsImpl) Stop() {
	close(h.stopSignal)
}

func (h *vulnerabilityMetricsImpl) run() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	ctx, cancel := context.WithCancel(context.Background())
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
