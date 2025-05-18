package common

import (
	"context"
	"iter"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/storage"
)

// TrackerConfig wraps various pieces of configuration required for tracking
// various metrics.
type TrackerConfig struct {
	category    string
	description string
	labelOrder  map[Label]int
	gatherFunc  func(context.Context) iter.Seq[func(Label) string]

	// metricsConfig can be changed with an API call.
	metricsConfig    MetricLabelExpressions
	metricsConfigMux sync.RWMutex

	// periodCh allows for changing the period in runtime.
	periodCh chan time.Duration
}

func MakeTrackerConfig(category, description string, labelOrder map[Label]int, gatherFunc func(context.Context) iter.Seq[func(Label) string]) *TrackerConfig {
	return &TrackerConfig{
		category:    category,
		description: description,
		labelOrder:  labelOrder,
		gatherFunc:  gatherFunc,

		periodCh: make(chan time.Duration, 1),
	}
}

func (tc *TrackerConfig) GetPeriodCh() <-chan time.Duration {
	return tc.periodCh
}

func (tc *TrackerConfig) Reconfigure(registry *prometheus.Registry, cfg map[string]*storage.PrometheusMetricsConfig_LabelExpressions, period time.Duration) error {
	mle, err := parseMetricLabels(cfg, tc.labelOrder)
	if err != nil {
		log.Errorf("Failed to parse metrics configuration for %s: %v", tc.category, err)
		return err
	}
	tc.metricsConfigMux.Lock()
	defer tc.metricsConfigMux.Unlock()
	tc.metricsConfig = mle
	select {
	case tc.periodCh <- period:
		break
	default:
		// If the period has not been read, read it now:
		<-tc.periodCh
		tc.periodCh <- period
	}
	registerMetrics(registry, tc.category, tc.description, tc.labelOrder, mle, period)
	return nil
}

func (tc *TrackerConfig) GetMetricLabelExpressions() MetricLabelExpressions {
	tc.metricsConfigMux.RLock()
	defer tc.metricsConfigMux.RUnlock()
	return tc.metricsConfig
}
