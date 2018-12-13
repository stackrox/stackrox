package providermetadata

import (
	"sync"

	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
)

var (
	once sync.Once

	pi pipeline.Pipeline
)

func initialize() {
	pi = NewPipeline(datastore.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() pipeline.Pipeline {
	once.Do(initialize)
	return pi
}
