package namespaces

import (
	"sync"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	namespaceStore "github.com/stackrox/rox/central/namespace/store"
	"github.com/stackrox/rox/central/networkpolicies/graph"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
)

var (
	once sync.Once

	pi pipeline.Fragment
)

func initialize() {
	pi = NewPipeline(clusterDataStore.Singleton(), namespaceStore.Singleton(), graph.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() pipeline.Fragment {
	once.Do(initialize)
	return pi
}
