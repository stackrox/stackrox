package nodes

import (
	"sync"

	"github.com/stackrox/rox/central/node/store"
	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
)

var (
	once sync.Once

	nodesPipeline pipeline.Pipeline
)

func initialize() {
	nodesPipeline = NewPipeline(store.Singleton())
}

// Singleton provides the instance of the cluster status pipeline.
func Singleton() pipeline.Pipeline {
	once.Do(initialize)
	return nodesPipeline
}
