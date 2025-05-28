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
	Gather(context.Context) (map[string]any, error)
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
			iv.GetQuery(),
			iv.GetMetricLabels(),
			time.Hour*time.Duration(iv.GetGatheringPeriodHours())); err != nil {
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

func (ar *aggregatorRunner) Gather(ctx context.Context) (map[string]any, error) {
	systemPrivateConfig, err := configDS.Singleton().GetPrivateConfig(
		sac.WithAllAccess(ctx))
	if err != nil {
		return nil, err
	}
	props := make(map[string]any)
	cfg := systemPrivateConfig.GetPrometheusMetricsConfig()
	{
		vulns := cfg.GetImageVulnerabilities()
		ml := vulns.GetMetricLabels()
		props["Total Image Vulnerability custom metrics"] = len(ml)
		maxLabels := 0
		exressionsUsed := false
		for _, metricLabels := range ml {
			if len(metricLabels.GetLabelExpressions()) > maxLabels {
				maxLabels = len(metricLabels.GetLabelExpressions())
			}
			if !exressionsUsed {
				for _, labelExprs := range metricLabels.GetLabelExpressions() {
					if len(labelExprs.GetExpression()) > 0 {
						exressionsUsed = true
						break
					}
				}
			}
		}
		props["Max Image Vulnerability custom metrics labels"] = maxLabels
		props["Custom Metrics Expressions used"] = exressionsUsed
	}
	return props, nil
}
