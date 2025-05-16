package aggregator

import (
	"sync"
	"time"
)

type tracker struct {
	category string

	metricsConfig    metricsConfig
	metricsConfigMux sync.RWMutex
	periodCh         chan time.Duration
}

func makeTracker(category string) *tracker {
	return &tracker{category: category}
}

func (mt *tracker) reloadConfig(cfg metricsConfig, period time.Duration) {
	mt.metricsConfigMux.Lock()
	defer mt.metricsConfigMux.Unlock()
	mt.metricsConfig = cfg
	if mt.periodCh == nil {
		mt.periodCh = make(chan time.Duration, 1)
	}
	select {
	case mt.periodCh <- period:
		break
	default:
		// If the period has not been read, read it now:
		<-mt.periodCh
		mt.periodCh <- period
	}
	registerMetrics(mt.category, cfg, period)
}

func (mt *tracker) getMetricsConfig() metricsConfig {
	if mt != nil {
		mt.metricsConfigMux.RLock()
		defer mt.metricsConfigMux.RUnlock()
		return mt.metricsConfig
	}
	return nil
}
