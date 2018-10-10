package processindicators

import (
	"sync"

	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/lifecycle"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
)

var (
	once sync.Once

	pi pipeline.Pipeline
)

func initialize() {
	pi = NewPipeline(processIndicatorDataStore.Singleton(), deploymentDataStore.Singleton(), lifecycle.SingletonManager())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() pipeline.Pipeline {
	once.Do(initialize)
	return pi
}
