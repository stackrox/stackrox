package all

import (
	"github.com/stackrox/stackrox/central/sensor/service/pipeline"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	factory pipeline.Factory
)

// Singleton provides the factory that creates pipelines per cluster.
func Singleton() pipeline.Factory {
	once.Do(func() {
		factory = NewFactory()
	})
	return factory
}
