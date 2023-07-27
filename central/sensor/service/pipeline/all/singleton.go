package all

import (
	hashManager "github.com/stackrox/rox/central/hash/manager"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	usageDS "github.com/stackrox/rox/central/usage/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	factory pipeline.Factory
)

// Singleton provides the factory that creates pipelines per cluster.
func Singleton() pipeline.Factory {
	once.Do(func() {
		factory = NewFactory(hashManager.Singleton(), usageDS.Singleton())
	})
	return factory
}
