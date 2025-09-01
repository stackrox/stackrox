package custom

import (
	"context"
	"net/http"

	configDS "github.com/stackrox/rox/central/config/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/metrics/custom/image_vulnerabilities"
	custom "github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
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
	image_vulnerabilities custom.Tracker
}

type Runner interface {
	http.Handler
	ValidateConfiguration(*storage.PrometheusMetrics) (*RunnerConfiguration, error)
	Reconfigure(*RunnerConfiguration)
}

// Singleton returns a runner, or nil if there were errors durig initialization.
// nil runner is safe, but no-op.
func Singleton() Runner {
	oneRunner.Do(func() {
		runner = makeRunner(metrics.MakeCustomRegistry(), deploymentDS.Singleton())
		go runner.initialize(configDS.Singleton())
	})
	return runner
}

func makeRunner(registry metrics.CustomRegistry, dds deploymentDS.DataStore) *aggregatorRunner {
	return &aggregatorRunner{
		Handler:               promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
		image_vulnerabilities: image_vulnerabilities.New(registry, dds),
	}
}

func (ar *aggregatorRunner) initialize(cds configDS.DataStore) {
	ctx := sac.WithAllAccess(context.Background())
	systemPrivateConfig, err := cds.GetPrivateConfig(ctx)
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
	image_vulnerabilities *custom.Configuration
}

func (ar *aggregatorRunner) ValidateConfiguration(cfg *storage.PrometheusMetrics) (*RunnerConfiguration, error) {
	if ar == nil {
		return &RunnerConfiguration{}, nil
	}
	var err error
	runnerConfig := &RunnerConfiguration{}
	runnerConfig.image_vulnerabilities, err = ar.image_vulnerabilities.NewConfiguration(cfg.GetImageVulnerabilities())
	if err != nil {
		return nil, err
	}
	return runnerConfig, nil
}

// Reconfigure will panic on nil cfg. Don't pass nil.
func (ar *aggregatorRunner) Reconfigure(cfg *RunnerConfiguration) {
	ar.image_vulnerabilities.Reconfigure(cfg.image_vulnerabilities)
}

func (ar *aggregatorRunner) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if id := authn.IdentityFromContextOrNil(req.Context()); id != nil {
		// The request context is cancelled when the client's connection closes.
		ctx := authn.CopyContextIdentity(context.Background(), req.Context())
		go ar.image_vulnerabilities.Gather(ctx)
	}
	ar.Handler.ServeHTTP(w, req)
}
