package networkflowupdate

import (
	"sync"

	"github.com/stackrox/rox/central/networkflow/store"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
)

var (
	once sync.Once

	pi pipeline.FragmentFactory
)

func initialize() {
	pi = NewFactory(store.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() pipeline.FragmentFactory {
	once.Do(initialize)
	return pi
}
