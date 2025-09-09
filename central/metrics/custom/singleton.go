package custom

import (
	"net/http"

	alertDS "github.com/stackrox/rox/central/alert/datastore"
	configDS "github.com/stackrox/rox/central/config/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	runner     *aggregatorRunner
	onceRunner sync.Once

	log = logging.LoggerForModule()
)

type Runner interface {
	http.Handler
	ValidateConfiguration(*storage.PrometheusMetrics) (*RunnerConfiguration, error)
	Reconfigure(*RunnerConfiguration)
}

// Singleton returns a runner, or nil if there were errors during
// initialization. nil runner is safe, but no-op.
func Singleton() Runner {
	onceRunner.Do(func() {
		runner = makeRunner(metrics.MakeCustomRegistry(), deploymentDS.Singleton(), alertDS.Singleton())
		go runner.initialize(configDS.Singleton())
	})
	return runner
}
