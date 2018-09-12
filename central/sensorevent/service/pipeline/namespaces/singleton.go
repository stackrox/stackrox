package namespaces

import (
	"sync"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	namespaceStore "github.com/stackrox/rox/central/namespace/store"
	"github.com/stackrox/rox/central/networkgraph"
	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
)

var (
	once sync.Once

	pi pipeline.Pipeline
)

func initialize() {
	pi = NewPipeline(clusterDataStore.Singleton(), namespaceStore.Singleton(), networkgraph.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() pipeline.Pipeline {
	once.Do(initialize)
	return pi
}
