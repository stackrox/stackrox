package aggregator

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	configDS "github.com/stackrox/rox/central/config/datastore"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/metrics/aggregator/common"
	"github.com/stackrox/rox/central/metrics/aggregator/vulnerabilities"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	Registry = prometheus.NewRegistry()

	once     sync.Once
	instance *aggregatorRunner

	log = logging.LoggerForModule()
)

type aggregatorRunner struct {
	stopCh      chan bool
	stopOnce    sync.Once
	trackersMux sync.RWMutex

	vulnerabilities     *common.TrackerConfig
	vulnerabilitiesOnce sync.Once
}

func Singleton() interface {
	Start()
	Stop()
	Reconfigure(*storage.PrometheusMetricsConfig) error
} {
	once.Do(func() {
		instance = &aggregatorRunner{
			stopCh:          make(chan bool),
			vulnerabilities: vulnerabilities.MakeTrackerConfig(),
		}
		systemPrivateConfig, err := configDS.Singleton().GetPrivateConfig(
			sac.WithAllAccess(context.Background()))
		if err != nil {
			log.Errorw("Failed to read Prometheus metrics configuration from the DB", logging.Err(err))
			return
		}
		if err := instance.Reconfigure(systemPrivateConfig.GetPrometheusMetricsConfig()); err != nil {
			log.Errorw("Failed to configure Prometheus metrics", logging.Err(err))
		}
	})
	return instance
}

func (ar *aggregatorRunner) Reconfigure(cfg *storage.PrometheusMetricsConfig) error {
	ar.trackersMux.Lock()
	defer ar.trackersMux.Unlock()

	{
		vc := cfg.GetVulnerabilities()
		period := time.Hour * time.Duration(vc.GetGatheringPeriodHours())
		err := instance.vulnerabilities.Reconfigure(Registry, vc.GetMetricLabels(), period)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ar *aggregatorRunner) Start() {
	ar.trackersMux.RLock()
	defer ar.trackersMux.RUnlock()

	// Run the periodic vulnerabilities aggregation.
	ar.vulnerabilitiesOnce.Do(func() {
		if ar.vulnerabilities != nil {
			vulnTracker := common.MakeTrackFunc(
				ar.vulnerabilities,
				metrics.SetCustomAggregatedCount,
			)
			go ar.run(ar.vulnerabilities.GetPeriodCh(), vulnTracker)
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
