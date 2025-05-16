package aggregator

import (
	"context"
	"time"

	configDS "github.com/stackrox/rox/central/config/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance *aggregatorRunner

	log = logging.LoggerForModule()
)

type aggregatorRunner struct {
	stopCh   chan bool
	stopOnce sync.Once
	mux      sync.RWMutex

	vulnerabilities     *tracker
	vulnerabilitiesOnce sync.Once
}

func Singleton() interface {
	Start()
	Stop()
	ReloadConfig(*storage.PrometheusMetricsConfig) error
} {
	once.Do(func() {
		instance = &aggregatorRunner{
			stopCh: make(chan bool),
		}
		systemPrivateConfig, err := configDS.Singleton().GetPrivateConfig(
			sac.WithAllAccess(context.Background()))
		if err != nil {
			log.Errorw("Failed to get Prometheus metrics configuration", logging.Err(err))
			return
		}
		// Ignore error on start, as there's nothing we can do.
		_ = instance.ReloadConfig(systemPrivateConfig.GetPrometheusMetricsConfig())
	})
	return instance
}

func (ar *aggregatorRunner) ReloadConfig(cfg *storage.PrometheusMetricsConfig) error {
	ar.mux.Lock()
	defer ar.mux.Unlock()

	// Vulnerabilties metrics:
	vulnTracker, err := reloadVulnerabilityTrackerConfig(cfg.GetVulnerabilities())
	if err == nil || instance.vulnerabilities == nil {
		instance.vulnerabilities = vulnTracker
	}
	if err != nil {
		return err
	}

	return nil
}

func (ar *aggregatorRunner) Start() {
	// Run the periodic vulnerabilities aggregation.
	ar.vulnerabilitiesOnce.Do(func() {
		if v := ar.vulnerabilities; v != nil {
			tw := makeTrackWrapper(
				deploymentDS.Singleton(),
				ar.vulnerabilities.getMetricsConfig,
				trackVulnerabilityMetrics)
			go ar.run(v.periodCh, tw.track)
		}
	})
}

func (ar *aggregatorRunner) Stop() {
	ar.stopOnce.Do(func() {
		close(ar.stopCh)
	})
}

func (ar *aggregatorRunner) run(periodCh <-chan time.Duration, track func(context.Context)) {
	var ticker *time.Ticker

	ctx, cancel := context.WithCancel(
		sac.WithAllAccess(context.Background()))
	defer cancel()

	for {
		select {
		case <-ticker.C:
			track(ctx)
		case <-ar.stopCh:
			return
		case period := <-periodCh:
			if period > 0 {
				track(ctx)
				if ticker == nil {
					ticker = time.NewTicker(period)
					defer ticker.Stop()
				} else {
					ticker.Reset(period)
				}
			} else {
				if ticker != nil {
					ticker.Stop()
				}
			}
		}
	}
}
