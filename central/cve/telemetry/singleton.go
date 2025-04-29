package telemetry

import (
	"context"
	"strings"
	"time"

	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/pkg/env"
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
		if env.AggregateVulnMetrics.Setting() == "" {
			return
		}
		metricExpressions := parseAggregationExpressions(env.AggregateVulnMetrics.Setting())
		if metricExpressions == nil {
			return
		}
		instance = &vulnerabilityMetricsImpl{
			ds:                deploymentDS.Singleton(),
			metricExpressions: metricExpressions,
			trackFunc:         metrics.SetAggregatedVulnCount,
		}
		for metricName, expressions := range instance.metricExpressions {
			labels := getMetricLabels(expressions)
			var expressionStrings []string
			for _, expr := range expressions {
				expressionStrings = append(expressionStrings, expr.String())
			}
			metrics.RegisterVulnAggregatedMetric(metricName, labels,
				strings.Join(expressionStrings, ","))
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
	metricExpressions map[metricName][]*expression
	trackFunc         func(metricName string, labels map[Label]string, total int)
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
	ticker := time.NewTicker(env.AggregateVulnMetricsPeriod.DurationSetting())
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
