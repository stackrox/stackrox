package custom

import (
	"net/http"

	alertDS "github.com/stackrox/rox/central/alert/datastore"
	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	configDS "github.com/stackrox/rox/central/config/datastore"
	expiryS "github.com/stackrox/rox/central/credentialexpiry/service"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	policyDS "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	runner     trackerRunner
	onceRunner sync.Once

	log = logging.LoggerForModule()
)

type Runner interface {
	http.Handler
	ValidateConfiguration(*storage.PrometheusMetrics) (RunnerConfiguration, error)
	Reconfigure(RunnerConfiguration)
}

// Singleton returns a runner, or nil if there were errors during
// initialization. nil runner is safe, but no-op.
func Singleton() Runner {
	onceRunner.Do(func() {
		runner = makeRunner(&runnerDatastores{
			deploymentDS.Singleton(),
			alertDS.Singleton(),
			clusterDS.Singleton(),
			policyDS.Singleton(),
			expiryS.Singleton(),
		})
		go runner.initialize(configDS.Singleton())
	})
	return runner
}
