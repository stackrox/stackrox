package processindicators

import (
	"sync"

	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/runtime"
	"github.com/stackrox/rox/central/detection/utils"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
)

var (
	once sync.Once

	pi pipeline.Pipeline
)

func initialize() {
	pi = NewPipeline(
		processIndicatorDataStore.Singleton(),
		runtime.SingletonPolicySet(),
		utils.SingletonAlertManager(),
		deploymentDataStore.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() pipeline.Pipeline {
	once.Do(initialize)
	return pi
}
