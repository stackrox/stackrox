package common

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// TrackerConfig wraps various pieces of configuration required for tracking
// various metrics.
type TrackerConfig struct {
	category    string
	description string
	labelOrder  map[Label]int

	// metricsConfig can be changed with an API call.
	metricsConfig    MetricLabelExpressions
	metricsConfigMux sync.RWMutex

	// periodCh allows for changing the period in runtime.
	periodCh chan time.Duration
}

func MakeTrackerConfig(category, description string, labelOrder map[Label]int) *TrackerConfig {
	return &TrackerConfig{
		category:    category,
		description: description,
		labelOrder:  labelOrder,
		periodCh:    make(chan time.Duration, 1),
	}
}

func (mt *TrackerConfig) GetPeriodCh() <-chan time.Duration {
	return mt.periodCh
}

func (mt *TrackerConfig) Reconfigure(registry *prometheus.Registry, mle MetricLabelExpressions, period time.Duration) {
	mt.metricsConfigMux.Lock()
	defer mt.metricsConfigMux.Unlock()
	mt.metricsConfig = mle
	select {
	case mt.periodCh <- period:
		break
	default:
		// If the period has not been read, read it now:
		<-mt.periodCh
		mt.periodCh <- period
	}
	registerMetrics(registry, mt.category, mt.description, mt.labelOrder, mle, period)
}

func (mt *TrackerConfig) GetMetricLabelExpressions() MetricLabelExpressions {
	mt.metricsConfigMux.RLock()
	defer mt.metricsConfigMux.RUnlock()
	return mt.metricsConfig
}
