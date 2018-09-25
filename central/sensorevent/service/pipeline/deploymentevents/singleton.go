package deploymentevents

import (
	"sync"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	deployTimeDetection "github.com/stackrox/rox/central/detection/deploytime"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/networkpolicies/graph"
	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
)

var (
	once sync.Once

	pi pipeline.Pipeline
)

func initialize() {
	pi = NewPipeline(clusterDataStore.Singleton(),
		deploymentDataStore.Singleton(),
		imageDataStore.Singleton(),
		deployTimeDetection.SingletonDetector(),
		graph.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() pipeline.Pipeline {
	once.Do(initialize)
	return pi
}
