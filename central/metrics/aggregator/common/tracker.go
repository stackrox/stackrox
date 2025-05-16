package common

import (
	"sync"
	"time"
)

type Tracker struct {
	category    string
	description string
	labelOrder  map[Label]int

	metricsConfig    MetricsConfig
	metricsConfigMux sync.RWMutex
	periodCh         chan time.Duration
}

func MakeTracker(category, description string, labelOrder map[Label]int) *Tracker {
	return &Tracker{category: category, description: description, labelOrder: labelOrder}
}

func (mt *Tracker) GetPeriodCh() <-chan time.Duration {
	return mt.periodCh
}

func (mt *Tracker) Reconfigure(cfg MetricsConfig, period time.Duration) {
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
	registerMetrics(mt.category, mt.description, mt.labelOrder, cfg, period)
}

func (mt *Tracker) GetMetricsConfig() MetricsConfig {
	if mt != nil {
		mt.metricsConfigMux.RLock()
		defer mt.metricsConfigMux.RUnlock()
		return mt.metricsConfig
	}
	return nil
}
