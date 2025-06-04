package aggregator

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	configDS "github.com/stackrox/rox/central/config/datastore"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/metrics/aggregator/common"
	"github.com/stackrox/rox/central/metrics/aggregator/image_vulnerabilities"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	Registry = prometheus.NewRegistry()

	once   sync.Once
	runner *aggregatorRunner

	log = logging.LoggerForModule()
)

type aggregatorRunner struct {
	stopCh   chan bool
	stopOnce sync.Once

	image_vulnerabilities common.Tracker
}

func Singleton() interface {
	Start()
	Stop()
	Reconfigure(*storage.PrometheusMetricsConfig) error
} {
	once.Do(func() {
		runner = &aggregatorRunner{
			stopCh:                make(chan bool),
			image_vulnerabilities: image_vulnerabilities.MakeTrackerConfig(metrics.SetCustomAggregatedCount),
		}
		systemPrivateConfig, err := configDS.Singleton().GetPrivateConfig(
			sac.WithAllAccess(context.Background()))
		if err != nil {
			log.Errorw("Failed to read Prometheus metrics configuration from the DB", logging.Err(err))
			return
		}
		if err := runner.Reconfigure(systemPrivateConfig.GetPrometheusMetricsConfig()); err != nil {
			log.Errorw("Failed to configure Prometheus metrics", logging.Err(err))
		}
	})
	return runner
}

func (ar *aggregatorRunner) Reconfigure(cfg *storage.PrometheusMetricsConfig) error {
	{
		iv := cfg.GetImageVulnerabilities()
		if err := ar.image_vulnerabilities.Reconfigure(Registry,
			iv.GetFilter(),
			iv.GetMetrics(),
			time.Minute*time.Duration(iv.GetGatheringPeriodMinutes())); err != nil {
			return err
		}
	}
	return nil
}

func (ar *aggregatorRunner) Start() {
	ar.image_vulnerabilities.Do(func() {
		go ar.run(ar.image_vulnerabilities)
	})
}

func (ar *aggregatorRunner) Stop() {
	ar.stopOnce.Do(func() {
		close(ar.stopCh)
	})
}

func (ar *aggregatorRunner) run(tracker common.Tracker) {
	periodCh := tracker.GetPeriodCh()
	// The ticker will be reset immediately when reading from the periodCh.
	ticker := time.NewTicker(1000 * time.Hour)
	defer ticker.Stop()

	ctx, cancel := context.WithCancel(
		sac.WithAllAccess(context.Background()))
	defer cancel()

	for {
		select {
		case <-ticker.C:
			tracker.Track(ctx)
		case <-ar.stopCh:
			return
		case period := <-periodCh:
			if period > 0 {
				tracker.Track(ctx)
				ticker.Reset(period)
			} else {
				ticker.Stop()
			}
		}
	}
}
