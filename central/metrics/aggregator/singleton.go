package aggregator

import (
	"context"
	"time"

	configDS "github.com/stackrox/rox/central/config/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/metrics/aggregator/common"
	"github.com/stackrox/rox/central/metrics/aggregator/vulnerabilities"
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

	vulnerabilities     *common.Tracker
	vulnerabilitiesOnce sync.Once
}

func Singleton() interface {
	Start()
	Stop()
	Reconfigure(*storage.PrometheusMetricsConfig) error
} {
	once.Do(func() {
		instance = &aggregatorRunner{
			stopCh: make(chan bool),
		}
		systemPrivateConfig, err := configDS.Singleton().GetPrivateConfig(
			sac.WithAllAccess(context.Background()))
		if err != nil {
			log.Errorw("Failed to read Prometheus metrics configuration from the DB", logging.Err(err))
			return
		}
		if err := instance.Reconfigure(systemPrivateConfig.GetPrometheusMetricsConfig()); err != nil {
			log.Errorw("Failed to initialize Prometheus metrics configuration", logging.Err(err))
		}
	})
	return instance
}

func (ar *aggregatorRunner) Reconfigure(cfg *storage.PrometheusMetricsConfig) error {
	ar.mux.Lock()
	defer ar.mux.Unlock()

	// Vulnerabilties metrics:
	vulnTracker, err := vulnerabilities.Reconfigure(cfg.GetVulnerabilities())
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
		if vulnTracker := ar.vulnerabilities; vulnTracker != nil {
			tw := common.MakeTrackWrapper(
				deploymentDS.Singleton(),
				vulnTracker.GetMetricsConfig,
				vulnerabilities.TrackVulnerabilityMetrics)
			go ar.run(vulnTracker.GetPeriodCh(), tw.Track)
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
