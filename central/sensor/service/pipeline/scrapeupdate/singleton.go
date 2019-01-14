package scrapeupdate

import (
	"sync"

	"github.com/stackrox/rox/central/sensor/service/pipeline"
)

var (
	once sync.Once

	pi pipeline.Fragment
)

// Singleton provides the instance of the Service interface to register.
func Singleton() pipeline.Fragment {
	once.Do(func() {
		pi = NewPipeline()
	})
	return pi
}
