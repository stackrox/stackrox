package lifecycle

import (
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/activecomponent/updater/aggregator"
	"github.com/stackrox/stackrox/central/deployment/cache"
	deploymentDatastore "github.com/stackrox/stackrox/central/deployment/datastore"
	"github.com/stackrox/stackrox/central/detection/alertmanager"
	"github.com/stackrox/stackrox/central/detection/deploytime"
	"github.com/stackrox/stackrox/central/detection/runtime"
	policyDataStore "github.com/stackrox/stackrox/central/policy/datastore"
	baselineDataStore "github.com/stackrox/stackrox/central/processbaseline/datastore"
	processDatastore "github.com/stackrox/stackrox/central/processindicator/datastore"
	"github.com/stackrox/stackrox/central/processindicator/filter"
	"github.com/stackrox/stackrox/central/reprocessor"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/utils"
)

var (
	once    sync.Once
	manager *managerImpl
)

func initialize() {
	manager = newManager(
		deploytime.SingletonDetector(),
		runtime.SingletonDetector(),
		deploymentDatastore.Singleton(),
		processDatastore.Singleton(),
		baselineDataStore.Singleton(),
		alertmanager.Singleton(),
		reprocessor.Singleton(),
		cache.DeletedDeploymentCacheSingleton(),
		filter.Singleton(),
		aggregator.Singleton(),
	)

	policies, err := policyDataStore.Singleton().GetAllPolicies(lifecycleMgrCtx)
	utils.CrashOnError(err)
	log.Infof("Injecting %d policies into detectors.", len(policies))
	for _, policy := range policies {
		err = manager.UpsertPolicy(policy)
		utils.Should(errors.Wrap(err, "could not inject policy"))
	}
	log.Info("Done injecting policies.")

	go manager.buildIndicatorFilter()
}

// SingletonManager returns the manager instance.
func SingletonManager() Manager {
	once.Do(initialize)
	return manager
}
