package aggregator

import (
	"context"
	"net/http"

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

	ctx    context.Context
	cancel context.CancelFunc

	image_vulnerabilities common.Tracker
}

type Runner interface {
	http.Handler
	Start()
	Stop()
	ValidateConfiguration(*storage.PrometheusMetrics) (*RunnerConfiguration, error)
	Reconfigure(*RunnerConfiguration)
}

// Singleton returns a runner, or nil if there were errors durig initialization.
// nil runner is safe, but no-op.
func Singleton() Runner {
	oneRunner.Do(func() {
		runner = makeRunner(deploymentDS.Singleton())
		go runner.initialize(configDS.Singleton())
	})
	return runner
}

func makeRunner(dds deploymentDS.DataStore) *aggregatorRunner {
	ar := &aggregatorRunner{
		Handler:               promhttp.HandlerFor(metrics.CustomRegistry, promhttp.HandlerOpts{}),
		image_vulnerabilities: image_vulnerabilities.New(metrics.SetCustomAggregatedCount, dds),
	}
	ar.ctx, ar.cancel = context.WithCancel(sac.WithAllAccess(context.Background()))

	return ar
}

func (ar *aggregatorRunner) initialize(cds configDS.DataStore) {
	systemPrivateConfig, err := cds.GetPrivateConfig(ar.ctx)
	if err != nil {
		log.Errorw("Failed to read Prometheus metrics configuration from the DB", logging.Err(err))
		return
	}

	cfg, err := ar.ValidateConfiguration(systemPrivateConfig.GetMetrics())
	if err != nil {
		log.Errorw("Failed to configure Prometheus metrics", logging.Err(err))
		return
	}
	ar.Reconfigure(cfg)
}

// RunnerConfiguration is to pass between ParseConfiguration and Reconfigure.
type RunnerConfiguration struct {
	image_vulnerabilities *common.Configuration
}

func (ar *aggregatorRunner) ValidateConfiguration(cfg *storage.PrometheusMetrics) (*RunnerConfiguration, error) {
	if ar == nil {
		return &RunnerConfiguration{}, nil
	}
	var err error
	runnerConfig := &RunnerConfiguration{}
	runnerConfig.image_vulnerabilities, err = ar.image_vulnerabilities.ValidateConfiguration(cfg.GetImageVulnerabilities())
	if err != nil {
		return nil, err
	}
	return runnerConfig, nil
}

// Reconfigure will panic on nil cfg. Don't pass nil.
func (ar *aggregatorRunner) Reconfigure(cfg *RunnerConfiguration) {
	ar.image_vulnerabilities.Reconfigure(ar.ctx, cfg.image_vulnerabilities)
}

func (ar *aggregatorRunner) Start() {
	for _, tracker := range []common.Tracker{ar.image_vulnerabilities} {
		go tracker.Run(ar.ctx)
	}
}

func (ar *aggregatorRunner) Stop() {
	ar.cancel()
}
