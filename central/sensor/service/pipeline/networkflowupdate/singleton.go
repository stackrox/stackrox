package networkflowupdate

import (
	networkFlowStoreSingleton "github.com/stackrox/rox/central/networkflow/store/singleton"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	pi pipeline.FragmentFactory
)

func initialize() {
	pi = NewFactory(networkFlowStoreSingleton.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() pipeline.FragmentFactory {
	once.Do(initialize)
	return pi
}
