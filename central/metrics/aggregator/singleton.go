package aggregator

import (
	"context"
	"net/http"
	"strings"

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
	oneRunner sync.Once

	log = logging.LoggerForModule()
)

type aggregatorRunner struct {
	http.Handler
	handlers    map[string]http.Handler
	handlersMux sync.Mutex

	ctx    context.Context
	cancel context.CancelFunc

	image_vulnerabilities common.Tracker
}

type Runner interface {
	http.Handler
	Start()
	Stop()
	ParseConfiguration(*storage.PrometheusMetricsConfig) (*RunnerConfiguration, error)
	Reconfigure(*RunnerConfiguration)
}

func Singleton() Runner {
	oneRunner.Do(func() {
		runner = makeRunner(configDS.Singleton(), deploymentDS.Singleton())
	})
	return runner
}

func makeRunner(ds configDS.DataStore, dds deploymentDS.DataStore) *aggregatorRunner {
	ar := &aggregatorRunner{
		handlers: map[string]http.Handler{},
	}

	systemPrivateConfig, err := ds.GetPrivateConfig(
		sac.WithAllAccess(context.Background()))
	if err != nil {
		log.Errorw("Failed to read Prometheus metrics configuration from the DB", logging.Err(err))
		return ar
	}

	ar.image_vulnerabilities = image_vulnerabilities.New(metrics.SetCustomAggregatedCount, dds)
	ar.ctx, ar.cancel = context.WithCancel(sac.WithAllAccess(context.Background()))

	cfg, err := ar.ParseConfiguration(systemPrivateConfig.GetPrometheusMetricsConfig())
	if err != nil {
		log.Errorw("Failed to configure Prometheus metrics", logging.Err(err))
	} else {
		ar.Reconfigure(cfg)
	}
	return ar
}

// RunnerConfiguration is to pass between ParseConfiguration and Reconfigure.
type RunnerConfiguration struct {
	image_vulnerabilities *common.Configuration
}

func (ar *aggregatorRunner) ParseConfiguration(cfg *storage.PrometheusMetricsConfig) (*RunnerConfiguration, error) {
	if ar == nil {
		return nil, nil
	}
	var err error
	runnerConfig := &RunnerConfiguration{}
	runnerConfig.image_vulnerabilities, err = ar.image_vulnerabilities.ParseConfiguration(cfg.GetImageVulnerabilities())
	if err != nil {
		return nil, err
	}
	return runnerConfig, nil
}

func (ar *aggregatorRunner) Reconfigure(cfg *RunnerConfiguration) {
	if ar == nil {
		return
	}
	ar.image_vulnerabilities.Reconfigure(ar.ctx, cfg.image_vulnerabilities)
}

func (ar *aggregatorRunner) Start() {
	if ar == nil {
		return
	}
	for _, tracker := range []common.Tracker{ar.image_vulnerabilities} {
		go tracker.Run(ar.ctx)
	}
}

func (ar *aggregatorRunner) Stop() {
	if ar == nil {
		return
	}
	ar.cancel()
}
func (ar *aggregatorRunner) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if ar == nil {
		w.WriteHeader(http.StatusOK)
	}

	if h := ar.getHandler(req); h != nil {
		h.ServeHTTP(w, req)
	} else {
		// Serve empty OK for unknown registry names.
		w.WriteHeader(http.StatusOK)
	}
}

func (ar *aggregatorRunner) getHandler(req *http.Request) http.Handler {
	registryName, ok := ar.getRegistryName(req)
	if !ok {
		return nil
	}
	registry := metrics.GetExternalRegistry(registryName)
	ar.handlersMux.Lock()
	defer ar.handlersMux.Unlock()
	h, ok := ar.handlers[registryName]
	if !ok {
		h = promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
		ar.handlers[registryName] = h
	}
	return h
}

func (*aggregatorRunner) getRegistryName(req *http.Request) (string, bool) {
	registryName, ok := strings.CutPrefix(req.URL.Path, "/metrics")
	if ok && (registryName == "" || strings.HasPrefix(registryName, "/")) {
		registryName = strings.TrimPrefix(registryName, "/")
		if metrics.IsKnownRegistry(registryName) {
			return registryName, true
		}
	}
	return "", false
}
