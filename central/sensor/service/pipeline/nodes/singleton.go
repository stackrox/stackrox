package nodes

import (
	"sync"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	nodeStore "github.com/stackrox/rox/central/node/globalstore"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
)

var (
	once sync.Once

	nodesPipeline pipeline.Fragment
)

func initialize() {
	nodesPipeline = NewPipeline(clusterDataStore.Singleton(), nodeStore.Singleton())
}

// Singleton provides the instance of the cluster status pipeline.
func Singleton() pipeline.Fragment {
	once.Do(initialize)
	return nodesPipeline
}
