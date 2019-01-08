package networkpolicies

import (
	"sync"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/networkpolicies/graph"
	networkPolicyStore "github.com/stackrox/rox/central/networkpolicies/store"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
)

var (
	once sync.Once

	pi pipeline.Fragment
)

func initialize() {
	pi = NewPipeline(clusterDataStore.Singleton(), networkPolicyStore.Singleton(), graph.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() pipeline.Fragment {
	once.Do(initialize)
	return pi
}
