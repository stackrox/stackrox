package aggregator

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	configDS "github.com/stackrox/rox/central/config/datastore"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/metrics/aggregator/common"
	"github.com/stackrox/rox/central/metrics/aggregator/image_vulnerabilities"
	"github.com/stackrox/rox/central/metrics/aggregator/node_vulnerabilities"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/travelaudience/go-promhttp"
)

var (
	once   sync.Once
	runner *aggregatorRunner

	log = logging.LoggerForModule()
)

type aggregatorRunner struct {
	http.Handler
	registry *prometheus.Registry

	stopCh   chan bool
	stopOnce sync.Once

	image_vulnerabilities common.Tracker
	node_vulnerabilities  common.Tracker
}

func Singleton() interface {
	http.Handler
	Start()
	Stop()
	Reconfigure(*storage.PrometheusMetricsConfig) error
} {
	once.Do(func() {
		registry := prometheus.NewRegistry()

		runner = &aggregatorRunner{
			Handler:  promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
			registry: registry,
			stopCh:   make(chan bool),

			image_vulnerabilities: image_vulnerabilities.MakeTrackerConfig(metrics.SetCustomAggregatedCount),
			node_vulnerabilities:  node_vulnerabilities.MakeTrackerConfig(metrics.SetCustomAggregatedCount),
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
		if err := ar.image_vulnerabilities.Reconfigure(ar.registry,
			iv.GetFilter(),
			iv.GetMetrics(),
			time.Minute*time.Duration(iv.GetGatheringPeriodMinutes())); err != nil {
			return err
		}
	}
	{
		nv := cfg.GetNodeVulnerabilities()
		if err := ar.node_vulnerabilities.Reconfigure(ar.registry,
			nv.GetFilter(),
			nv.GetMetrics(),
			time.Minute*time.Duration(nv.GetGatheringPeriodMinutes())); err != nil {
			return err
		}
	}
	return nil
}

func (ar *aggregatorRunner) Start() {
	for _, tracker := range []common.Tracker{ar.image_vulnerabilities, ar.node_vulnerabilities} {
		tracker.Do(func() {
			go ar.run(tracker)
		})
	}
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
