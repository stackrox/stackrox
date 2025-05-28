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
	"github.com/stackrox/rox/central/telemetry/centralclient"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
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
		handlers:              map[string]http.Handler{},
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

	cfg, err := ar.ParseConfiguration(systemPrivateConfig.GetPrometheusMetricsConfig())
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

func (ar *aggregatorRunner) ParseConfiguration(cfg *storage.PrometheusMetricsConfig) (*RunnerConfiguration, error) {
	if ar == nil {
		return &RunnerConfiguration{}, nil
	}
	var err error
	runnerConfig := &RunnerConfiguration{}
	runnerConfig.image_vulnerabilities, err = ar.image_vulnerabilities.ParseConfiguration(cfg.GetImageVulnerabilities())
	if err != nil {
		return nil, err
	}
	return runnerConfig, nil
}

// Reconfigure will panic on nil cfg. Don't pass nil.
func (ar *aggregatorRunner) Reconfigure(cfg *RunnerConfiguration) {
	ar.image_vulnerabilities.Reconfigure(ar.ctx, cfg.image_vulnerabilities)
	track(cfg)
}

func (ar *aggregatorRunner) Start() {
	for _, tracker := range []common.Tracker{ar.image_vulnerabilities} {
		go tracker.Run(ar.ctx)
	}
}

func (ar *aggregatorRunner) Stop() {
	ar.cancel()
}

func (ar *aggregatorRunner) ServeHTTP(w http.ResponseWriter, req *http.Request) {
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

func track(cfg *RunnerConfiguration) {
	if cfg == nil {
		return
	}
	centralclient.InstanceConfig().Telemeter().Track(
		"Prometheus metrics configured", nil,
		telemeter.WithTraits(makeProps(cfg)))
}

func makeProps(cfg *RunnerConfiguration) map[string]any {
	props := make(map[string]any, 3)
	{
		metrics := cfg.image_vulnerabilities.GetMetrics()
		labels, operators := getLabelsAndOps(metrics)
		props["Total Image Vulnerability metrics"] = len(metrics)
		props["Image Vulnerability metric labels"] = labels.AsSlice()
		props["Image Vulnerability metric operators"] = operators.AsSlice()
	}
	return props
}

func getLabelsAndOps(metrics common.MetricsConfiguration) (set.Set[string], set.Set[string]) {
	labels := set.NewStringSet()
	operators := set.NewStringSet()
	for _, metricLabels := range metrics {
		for label, labelExpr := range metricLabels {
			labels.Add(string(label))
			for _, condition := range labelExpr {
				operators.Add(condition.GetOperator())
			}
		}
	}
	return labels, operators
}
