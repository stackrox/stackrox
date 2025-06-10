package aggregator

import (
	"context"
	"net/http"
	"strings"
	"time"

	configDS "github.com/stackrox/rox/central/config/datastore"
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
	once   sync.Once
	runner *aggregatorRunner

	log = logging.LoggerForModule()
)

type aggregatorRunner struct {
	http.Handler

	handlers    map[string]http.Handler
	handlersMux sync.Mutex

	ctx      context.Context
	cancel   context.CancelFunc
	stopOnce sync.Once

	image_vulnerabilities common.Tracker
}

func Singleton() interface {
	http.Handler
	Start()
	Stop()
	Reconfigure(*storage.PrometheusMetricsConfig) error
} {
	once.Do(func() {

		runner = &aggregatorRunner{
			handlers:              map[string]http.Handler{},
			image_vulnerabilities: image_vulnerabilities.MakeTrackerConfig(metrics.SetCustomAggregatedCount),
		}

		runner.ctx, runner.cancel = context.WithCancel(sac.WithAllAccess(context.Background()))

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

func (ar *aggregatorRunner) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ar.handlersMux.Lock()
	defer ar.handlersMux.Unlock()

	registryName, _ := strings.CutPrefix(req.URL.Path, "/metrics/")
	registry := metrics.GetExternalRegistry(registryName)
	h, ok := ar.handlers[registryName]
	if !ok {
		h = promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
		ar.handlers[registryName] = h
	}
	h.ServeHTTP(w, req)
}

func (ar *aggregatorRunner) Reconfigure(cfg *storage.PrometheusMetricsConfig) error {
	{
		iv := cfg.GetImageVulnerabilities()
		if err := ar.image_vulnerabilities.Reconfigure(ar.ctx,
			iv.GetFilter(),
			iv.GetMetrics(),
			time.Minute*time.Duration(iv.GetGatheringPeriodMinutes())); err != nil {
			return err
		}
	}
	return nil
}

func (ar *aggregatorRunner) Start() {
	for _, tracker := range []common.Tracker{ar.image_vulnerabilities} {
		go tracker.Run(ar.ctx)
	}
}

func (ar *aggregatorRunner) Stop() {
	ar.cancel()
}
