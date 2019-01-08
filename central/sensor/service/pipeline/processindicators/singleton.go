package processindicators

import (
	"sync"

	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/lifecycle"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
)

var (
	once sync.Once

	pi pipeline.Fragment
)

func initialize() {
	pi = NewPipeline(processIndicatorDataStore.Singleton(), deploymentDataStore.Singleton(), lifecycle.SingletonManager())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() pipeline.Fragment {
	once.Do(initialize)
	return pi
}
