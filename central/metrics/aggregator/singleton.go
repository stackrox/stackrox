package aggregator

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	configDS "github.com/stackrox/rox/central/config/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/metrics/aggregator/common"
	"github.com/stackrox/rox/central/metrics/aggregator/image_vulnerabilities"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/travelaudience/go-promhttp"
)

var (
	runner    *aggregatorRunner
	runnerMux sync.RWMutex

	log = logging.LoggerForModule()
)

type aggregatorRunner struct {
	http.Handler
	registry *prometheus.Registry

	ctx    context.Context
	cancel context.CancelFunc

	image_vulnerabilities common.Tracker
}

type Runner interface {
	http.Handler
	Start()
	Stop()
	Reconfigure(*storage.PrometheusMetricsConfig) error
}

func Singleton() Runner {
	r := getRunner()
	if r != nil {
		return r
	}

	r = makeRunner(configDS.Singleton(), deploymentDS.Singleton())
	runnerMux.Lock()
	defer runnerMux.Unlock()
	runner = r
	return r
}

func getRunner() *aggregatorRunner {
	runnerMux.RLock()
	defer runnerMux.RUnlock()
	return runner
}

func makeRunner(ds configDS.DataStore, dds deploymentDS.DataStore) *aggregatorRunner {
	registry := prometheus.NewRegistry()

	r := &aggregatorRunner{
		Handler:  promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
		registry: registry,
	}

	systemPrivateConfig, err := ds.GetPrivateConfig(
		sac.WithAllAccess(context.Background()))
	if err != nil {
		log.Errorw("Failed to read Prometheus metrics configuration from the DB", logging.Err(err))
		return r
	}

	r.image_vulnerabilities = image_vulnerabilities.MakeTrackerConfig(metrics.SetCustomAggregatedCount, dds)
	r.ctx, r.cancel = context.WithCancel(sac.WithAllAccess(context.Background()))

	if err := r.Reconfigure(systemPrivateConfig.GetPrometheusMetricsConfig()); err != nil {
		log.Errorw("Failed to configure Prometheus metrics", logging.Err(err))
	}
	return r
}

func (ar *aggregatorRunner) Reconfigure(cfg *storage.PrometheusMetricsConfig) error {
	errs := []error{}
	{
		iv := cfg.GetImageVulnerabilities()
		errs = append(errs, ar.image_vulnerabilities.Reconfigure(ar.ctx,
			ar.registry,
			iv.GetFilter(),
			iv.GetMetrics(),
			time.Minute*time.Duration(iv.GetGatheringPeriodMinutes())))
	}
	return errors.Join(errs...)
}

func (ar *aggregatorRunner) Start() {
	for _, tracker := range []common.Tracker{ar.image_vulnerabilities} {
		go tracker.Run(ar.ctx)
	}
}

func (ar *aggregatorRunner) Stop() {
	ar.cancel()
}
