package deploymentevents

import (
	"sync"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/lifecycle"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/networkpolicies/graph"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
)

var (
	once sync.Once

	pi pipeline.Fragment
)

func initialize() {
	pi = NewPipeline(clusterDataStore.Singleton(),
		deploymentDataStore.Singleton(),
		imageDataStore.Singleton(),
		lifecycle.SingletonManager(),
		graph.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() pipeline.Fragment {
	once.Do(initialize)
	return pi
}
