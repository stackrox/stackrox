package compliancereturn

import (
	"sync"

	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
)

var (
	once sync.Once

	pi pipeline.Pipeline
)

// Singleton provides the instance of the Service interface to register.
func Singleton() pipeline.Pipeline {
	once.Do(func() {
		pi = NewPipeline()
	})
	return pi
}
