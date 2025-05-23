package aggregator

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	once   sync.Once
	runner *aggregatorRunner

	log = logging.LoggerForModule()
)

type aggregatorRunner struct {
	stopCh   chan bool
	stopOnce sync.Once
	needSac  atomic.Bool

	vulnerabilities common.Tracker
}

func Singleton() interface {
	Start()
	Stop()
	Reconfigure(*storage.PrometheusMetricsConfig) error
	http.Handler
} {
	once.Do(func() {
		runner = &aggregatorRunner{
			stopCh:          make(chan bool),
			vulnerabilities: vulnerabilities.MakeTrackerConfig(metrics.SetCustomAggregatedCount),
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

func vulnerabilitiesConfig(cfg *storage.PrometheusMetricsConfig) (map[string]*storage.PrometheusMetricsConfig_MetricLabels, time.Duration) {
	vc := cfg.GetVulnerabilities()
	period := time.Hour * time.Duration(vc.GetGatheringPeriodHours())
	return vc.GetMetricLabels(), period
}

func isSacNeeded(mle map[string]*storage.PrometheusMetricsConfig_MetricLabels) bool {
	for _, le := range mle {
		for label := range le.LabelExpressions {
			if label == "Cluster" || label == "Namespace" {
				return true
			}
		}
	}
	return false
}

func (ar *aggregatorRunner) Reconfigure(cfg *storage.PrometheusMetricsConfig) error {
	{
		mle, period := vulnerabilitiesConfig(cfg)
		ar.needSac.CompareAndSwap(false, isSacNeeded(mle))
		if err := ar.vulnerabilities.Reconfigure(Registry, mle, period); err != nil {
			return err
		}
	}
	return nil
}

func (ar *aggregatorRunner) Start() {
	ar.vulnerabilities.Do(func() {
		go ar.run(ar.vulnerabilities)
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

func (ar *aggregatorRunner) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var gatherer prometheus.Gatherer = Registry
	if ar.needSac.Load() {
		var err error
		log.Info("Serving metrics with SAC gatherer")
		gatherer, err = common.MakeSacGatherer(r.Context(), Registry)
		if err != nil {
			http.Error(w, "Failed to create metrics gatherer: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{}).ServeHTTP(w, r)
}
