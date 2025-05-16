package aggregator

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	configDS "github.com/stackrox/rox/central/config/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once        sync.Once
	instance    *aggregatorRunner
	instanceMux sync.RWMutex

	log           = logging.LoggerForModule()
	Problemetrics = prometheus.NewRegistry()
)

type aggregatorRunner struct {
	stopCh          chan bool
	vulnerabilities *vulnerabilityMetricsTracker
}

func Singleton() interface {
	Start()
	Stop()
} {
	once.Do(func() {
		systemPrivateConfig, err := configDS.Singleton().GetPrivateConfig(
			sac.WithAllAccess(context.Background()))
		if err != nil {
			log.Errorw("Failed to get Prometheus metrics configuration", logging.Err(err))
			return
		}
		_ = ReloadConfig(systemPrivateConfig.GetPrometheusMetricsConfig())
	})
	instanceMux.RLock()
	defer instanceMux.RUnlock()
	return instance
}

func ReloadConfig(cfg *storage.PrometheusMetricsConfig) error {
	instanceMux.Lock()
	defer instanceMux.Unlock()

	if instance == nil {
		instance = &aggregatorRunner{
			stopCh: make(chan bool),
		}
	}

	if instance.vulnerabilities == nil {
		instance.vulnerabilities = makeVulnerabilitiesTracker()
	}

	return instance.vulnerabilities.reloadConfig(cfg.GetVulnerabilities())
}

func (ar *aggregatorRunner) Start() {
	if ar != nil {
		v := ar.vulnerabilities
		go ar.run(v.periodCh,
			v.aggregator.getTracker(v.getMetricsConfig))
	}
}

func (ar *aggregatorRunner) Stop() {
	if ar != nil {
		close(ar.stopCh)
	}
}

func (ar *aggregatorRunner) run(periodCh <-chan time.Duration, track func(context.Context)) {
	ticker := time.NewTicker(<-periodCh)
	defer ticker.Stop()

	ctx, cancel := context.WithCancel(
		sac.WithAllAccess(context.Background()))

	track(ctx)
	defer cancel()
	for {
		select {
		case <-ticker.C:
			track(ctx)
		case <-ar.stopCh:
			return
		case period := <-periodCh:
			if period > 0 {
				ticker.Reset(period)
			} else {
				ticker.Stop()
			}
		}
	}
}
